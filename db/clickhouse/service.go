package clickhouse

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/pkg/errors"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

type ClickhouseInstance string

const (
	ClickhouseLocalInstance      ClickhouseInstance = "clickhouse-local"
	ClickhouseReplicatedInstance ClickhouseInstance = "clickhouse-replicated"
)

var DefaultClickhouseConnectionDetails = &config.DatabaseDetails{
	Driver:   "clickhouse-local",
	Address:  "127.0.0.1:9000",
	User:     "username",
	Password: "password",
	Database: "akai_test",
	Meter:    otel.GetMeterProvider().Meter("clickhouse_db"),
}

var MaxFlushInterval = 1 * time.Second

type ClickHouseDB struct {
	// Reference to the configuration
	conDetails   *config.DatabaseDetails
	instanceType ClickhouseInstance

	// Database handler
	// pool of connections per table for bulk inserts
	// the ch-driver is not threadsafe, but can still support 1 connection per table
	lowLevelConnPool map[string]*lowLevelConn
	lowPoolM         sync.RWMutex
	highLevelClient  driver.Conn // for side tasks, like Select and Delete
	highMu           sync.Mutex

	// pointers for all the query batchers
	qBatchers struct {
		blocks              *queryBatcher[models.Block]
		samplingItems       *queryBatcher[models.SamplingItem]
		sampleGenericVisits *queryBatcher[models.SampleGenericVisit]
		sampleValueVisits   *queryBatcher[models.SampleValueVisit]
		peerInfoVisits      *queryBatcher[models.PeerInfoVisit]
	}

	// caches to speedup querides
	currentNetwork config.Network

	// reference to all relevant db telemetry
	// telemetry *telemetry
}

func NewClickHouseDB(
	conDetails *config.DatabaseDetails,
	instance ClickhouseInstance,
	network config.Network) (*ClickHouseDB, error) {
	db := &ClickHouseDB{
		conDetails:       conDetails,
		instanceType:     instance,
		currentNetwork:   network,
		lowLevelConnPool: make(map[string]*lowLevelConn),
	}
	return db, nil
}

func (db *ClickHouseDB) Init(ctx context.Context, tableNames map[string]struct{}) error {
	// get the connectable drivers for the given tables to open further connections
	tableDrivers := db.getConnectableDrivers(tableNames)
	err := db.makeConnections(ctx, tableDrivers)
	if err != nil {
		return errors.Wrap(err, "connecting clickhouse db")
	}

	err = db.testConnection(ctx)
	if err != nil {
		return errors.Wrap(err, "testing clickhouse connection")
	}

	err = db.ensureMigrations(ctx)
	if err != nil {
		return errors.Wrap(err, "making clickhouse migrations")
	}

	// make a batcher for each of the subscribed tables
	err = db.composeBatchersForTables(tableNames)
	if err != nil {
		return errors.Wrap(err, "setting up batchers")
	}
	return nil
}

func (db *ClickHouseDB) Serve(ctx context.Context) error {
	go db.periodicBatchesFlusher(ctx, MaxFlushInterval)

	<-ctx.Done()
	log.Info("closing DB service")
	return db.Close(ctx)
}

func (db *ClickHouseDB) GetAllTables() map[string]struct{} {
	return map[string]struct{}{
		models.BlockTableName:              {},
		models.SamplingItemTableName:       {},
		models.SampleGenericVisitTableName: {},
		models.SampleValueVisitTableName:   {},
		models.PeerInfoVisitTableName:      {},
	}
}

func (db *ClickHouseDB) makeConnections(ctx context.Context, tableNames map[string]connectableTable) error {
	// TODO: create a new connection per tag and table
	for table, driver := range tableNames {
		hlog := log.WithFields(log.Fields{
			"table_name": driver.TableName(),
			"tag":        driver.Tag(),
		})
		// check if there is already a connection to that table
		db.lowPoolM.Lock()
		_, ok := db.lowLevelConnPool[table]
		db.lowPoolM.Unlock()
		if ok {
			hlog.Warnf("there was already a low level connection to the table")
		}
		// get low level connection
		tableConn, err := db.getLowLevelConnection(ctx, db.conDetails, driver.Tag())
		if err != nil {
			hlog.Errorf("getting low-level-conncetion: %s", err.Error())
			return err
		}
		// save the connection for the table
		db.lowPoolM.Lock()
		db.lowLevelConnPool[table] = tableConn
		db.lowPoolM.Unlock()
		hlog.Debug("successful connection to table...")
	}

	highCon, err := db.getHighLevelConnection(ctx, db.conDetails)
	if err != nil {
		return err
	}

	db.highMu.Lock()
	db.highLevelClient = highCon
	db.highMu.Unlock()
	return nil
}

func (db *ClickHouseDB) ensureMigrations(_ context.Context) error {
	return db.makeMigrations()
}

func (db *ClickHouseDB) testConnection(ctx context.Context) error {
	return db.highLevelClient.Ping(ctx)
}

func (db *ClickHouseDB) closeLowLevelConns() {
	db.lowPoolM.RLock()
	for tableName, conn := range db.lowLevelConnPool {
		err := conn.close()
		if err != nil {
			log.Errorf("closeing low-level-connection to %s table; %s", tableName, err.Error())
		}
	}
	db.lowPoolM.RUnlock()
}

// getTableDrivers returns a connectable interface from the driver of the give table name
func (db *ClickHouseDB) getConnectableDrivers(tables map[string]struct{}) map[string]connectableTable {
	drivers := make(map[string]connectableTable, 0)
	for table := range tables {
		var driver connectableTable
		switch table {
		case models.BlockTableName:
			driver = blockTableDriver

		case models.SamplingItemTableName:
			driver = samplingItemTableDriver

		case models.SampleGenericVisitTableName:
			driver = sampleGenericVisistsTableDriver

		case models.SampleValueVisitTableName:
			driver = sampleValueVisitTableDriver

		case models.PeerInfoVisitTableName:
			driver = peerInfoVisitsTableDriver

		default:
			log.Warnf("no driver found for table %s", table)
			continue
		}
		drivers[table] = driver
	}
	return drivers
}

// composeBatchersForTables
func (db *ClickHouseDB) composeBatchersForTables(tables map[string]struct{}) error {
	for table := range tables {
		switch table {

		case models.BlockTableName:
			driver := blockTableDriver
			batcher, err := newQueryBatcher[models.Block](driver, MaxFlushInterval, models.Block{}.BatchingSize())
			if err != nil {
				return err
			}
			db.qBatchers.blocks = batcher

		case models.SamplingItemTableName:
			driver := samplingItemTableDriver
			batcher, err := newQueryBatcher[models.SamplingItem](driver, MaxFlushInterval, models.SamplingItem{}.BatchingSize())
			if err != nil {
				return err
			}
			db.qBatchers.samplingItems = batcher

		case models.SampleGenericVisitTableName:
			driver := sampleGenericVisistsTableDriver
			batcher, err := newQueryBatcher[models.SampleGenericVisit](driver, MaxFlushInterval, models.SampleGenericVisit{}.BatchingSize())
			if err != nil {
				return err
			}
			db.qBatchers.sampleGenericVisits = batcher

		case models.SampleValueVisitTableName:
			driver := sampleValueVisitTableDriver
			batcher, err := newQueryBatcher[models.SampleValueVisit](driver, MaxFlushInterval, models.SampleValueVisit{}.BatchingSize())
			if err != nil {
				return err
			}
			db.qBatchers.sampleValueVisits = batcher

		case models.PeerInfoVisitTableName:
			driver := peerInfoVisitsTableDriver
			batcher, err := newQueryBatcher[models.PeerInfoVisit](driver, MaxFlushInterval, models.PeerInfoVisit{}.BatchingSize())
			if err != nil {
				return err
			}
			db.qBatchers.peerInfoVisits = batcher

		default:
			log.Warnf("no driver found for table %s", table)
			continue
		}
	}
	return nil
}

func (db *ClickHouseDB) periodicBatchesFlusher(ctx context.Context, interval time.Duration) {
	log.WithField("freq", interval).Info("clickhouse batch-flusher")
	flushT := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-flushT.C:
			err := db.flushAllBatchers(ctx)
			if err != nil {
				log.Panic("flushing batchers on periodic rutine", err)
			}
		}
	}
}

func (db *ClickHouseDB) flushAllBatchers(ctx context.Context) error {
	log.Debug("flushing all tables")
	// flush all the batchers
	err := db.flushBatcher(ctx, db.qBatchers.blocks)
	if err != nil {
		return err
	}
	err = db.flushBatcher(ctx, db.qBatchers.samplingItems)
	if err != nil {
		return err
	}
	err = db.flushBatcher(ctx, db.qBatchers.sampleGenericVisits)
	if err != nil {
		return err
	}
	err = db.flushBatcher(ctx, db.qBatchers.sampleValueVisits)
	if err != nil {
		return err
	}
	err = db.flushBatcher(ctx, db.qBatchers.peerInfoVisits)
	if err != nil {
		return err
	}

	return nil
}

func (db *ClickHouseDB) Close(ctx context.Context) error {
	err := db.flushAllBatchers(ctx)
	if err != nil {
		log.Errorf("flushing batchers %s", err.Error())
	}

	// close the connections with the db
	db.closeLowLevelConns()
	err = db.highLevelClient.Close()
	if err != nil {
		log.Errorf("closing high level connection - %s", err.Error())
	}

	log.Infof("connection to clichouse database closed...")
	return nil
}

func (db *ClickHouseDB) persistBatch(
	ctx context.Context,
	tag string,
	baseQuery string,
	tableName string,
	input proto.Input,
) error {
	hlog := log.WithFields(log.Fields{
		"module":    "clickhouse-db",
		"operation": "individual-persist",
		"tag":       tag,
	})

	// get the lowLevel connection to the table
	db.lowPoolM.RLock()
	tableConn, ok := db.lowLevelConnPool[tableName]
	db.lowPoolM.RUnlock()
	if !ok {
		log.Warnf("there was no previous connection with tag %s to table %s", tag, tableName)
		return fmt.Errorf("no connection to table %s", tableName)
	}

	tStart := time.Now()
	err := tableConn.persist(
		ctx,
		baseQuery,
		tableName,
		input,
	)
	elapsedTime := time.Since(tStart)
	if err != nil {
		hlog.Errorf("tx to clickhouse db %s", err.Error())
		return err
	}

	log.Debugf("query submitted in %s", elapsedTime)
	return nil
}

func (db *ClickHouseDB) flushBatcher(ctx context.Context, batcher flusheableBatcher) error {
	defer batcher.resetBatcher()
	persistable, itemsN := batcher.getPersistable()
	if itemsN <= 0 {
		return nil
	}

	log.WithFields(log.Fields{
		"table":      batcher.TableName(),
		"tag":        batcher.Tag(),
		"full":       batcher.isFull(),
		"flusheable": batcher.isFlusheable(),
		"items":      itemsN,
	}).Debug("flushing batcher...")

	err := db.persistBatch(
		ctx,
		batcher.Tag(),
		batcher.BaseQuery(),
		batcher.TableName(),
		persistable,
	)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("flushing %d itmes for table %s", batcher.currentLen(), batcher.TableName()))
	}
	return nil
}

func (db *ClickHouseDB) flushBatcherIfNeeded(ctx context.Context, batcher flusheableBatcher) error {
	if batcher.isFull() || batcher.isFlusheable() {
		db.flushBatcher(ctx, batcher)
	}
	return nil
}

func (db *ClickHouseDB) PersistNewBlock(ctx context.Context, block *models.Block) error {
	flushable := db.qBatchers.blocks.addItem(block)
	if flushable {
		err := db.flushBatcherIfNeeded(ctx, db.qBatchers.blocks)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *ClickHouseDB) GetAllBlocks(ctx context.Context) ([]models.Block, error) {
	db.highMu.Lock()
	defer db.highMu.Unlock()
	return requestAllBlocks(ctx, db.highLevelClient, db.currentNetwork.String())
}

func (db *ClickHouseDB) GetSampleableBlocks(ctx context.Context) ([]models.Block, error) {
	db.highMu.Lock()
	defer db.highMu.Unlock()
	return requestBlocksOnTTL(ctx, db.highLevelClient, db.currentNetwork.String())
}

func (db *ClickHouseDB) GetLastBlock(ctx context.Context) (models.Block, error) {
	db.highMu.Lock()
	defer db.highMu.Unlock()
	return requestLastBlock(ctx, db.highLevelClient)
}

func (db *ClickHouseDB) PersistNewSamplingItems(ctx context.Context, items []*models.SamplingItem) error {
	for _, item := range items {
		flushable := db.qBatchers.samplingItems.addItem(item)
		if flushable {
			err := db.flushBatcherIfNeeded(ctx, db.qBatchers.samplingItems)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (db *ClickHouseDB) PersistNewSamplingItem(ctx context.Context, item *models.SamplingItem) error {
	flushable := db.qBatchers.samplingItems.addItem(item)
	if flushable {
		err := db.flushBatcherIfNeeded(ctx, db.qBatchers.samplingItems)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *ClickHouseDB) GetSampleableItems(ctx context.Context) ([]models.SamplingItem, error) {
	db.highMu.Lock()
	defer db.highMu.Unlock()
	return requestItemsOnTTL(ctx, db.highLevelClient, db.currentNetwork.String())
}

func (db *ClickHouseDB) PersistNewSampleGenericVisit(ctx context.Context, visit *models.SampleGenericVisit) error {
	flushable := db.qBatchers.sampleGenericVisits.addItem(visit)
	if flushable {
		err := db.flushBatcherIfNeeded(ctx, db.qBatchers.sampleGenericVisits)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *ClickHouseDB) PersistNewSampleValueVisit(ctx context.Context, visit *models.SampleValueVisit) error {
	flushable := db.qBatchers.sampleValueVisits.addItem(visit)
	if flushable {
		err := db.flushBatcherIfNeeded(ctx, db.qBatchers.sampleValueVisits)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *ClickHouseDB) PersistNewPeerInfoVisit(ctx context.Context, visit *models.PeerInfoVisit) error {
	flushable := db.qBatchers.peerInfoVisits.addItem(visit)
	if flushable {
		err := db.flushBatcherIfNeeded(ctx, db.qBatchers.peerInfoVisits)
		if err != nil {
			return err
		}
	}
	return nil
}
