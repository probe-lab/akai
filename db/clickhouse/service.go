package clickhouse

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/pkg/errors"
	mdb "github.com/probe-lab/akai/db"
	log "github.com/sirupsen/logrus"

	lru "github.com/hashicorp/golang-lru"
)

type ClickHouseDB struct {
	// Reference to the configuration
	conDetails ConnectionDetails

	// Database handler
	// pool of connections per table for bulk inserts
	// the ch-driver is not threadsafe, but can still support 1 connection per table
	lowLevelConnPool map[string]*lowLevelConn
	lowPoolM         sync.RWMutex
	highLevelClient  driver.Conn // for side tasks, like Select and Delete
	highMu           sync.Mutex

	// pointers for all the query batchers
	qBatchers struct {
		network *queryBatcher[mdb.Network]
	}

	// caches to speedup queries
	networkIDs   *lru.Cache
	blockCellIDs *lru.Cache

	// reference to all relevant db telemetry
	// telemetry *telemetry
}

// var _ db.Database = (*ClickHouseDB)(nil)

func NewClickHouseDB(conDetails ConnectionDetails) (*ClickHouseDB, error) {
	db := &ClickHouseDB{
		conDetails:       conDetails,
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
	return db.composeBatchersForTables(tableNames)
}

func (db *ClickHouseDB) Run(context.Context) error {
	// TODO: generate all the batchers for the each tag and table
	// TODO: make the state machine from new events

	return nil
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
		hlog.Info("successful connection to table...")
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

func (db *ClickHouseDB) ensureMigrations(ctx context.Context) error {
	return db.makeMigrations(ctx)
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
		case mdb.NetworkTableName:
			driver = networkTableDriver

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
		case mdb.NetworkTableName:
			driver := networkTableDriver
			batcher, err := newQueryBatcher[mdb.Network](driver, mdb.Network{}.BatchingSize())
			if err != nil {
				return err
			}
			db.qBatchers.network = batcher

		default:
			log.Warnf("no driver found for table %s", table)
			continue
		}
	}
	return nil
}

func (db *ClickHouseDB) Close() {
	db.closeLowLevelConns()
	db.highLevelClient.Close()
	log.Infof("connection to clichouse database closed...")
}

func (db *ClickHouseDB) persistBatch(
	ctx context.Context,
	tag string,
	baseQuery string,
	tableName string,
	input proto.Input) error {

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

	log.Infof("query submitted in %s", elapsedTime)
	return nil
}
