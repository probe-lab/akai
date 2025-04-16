package core

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"

	log "github.com/sirupsen/logrus"
	"github.com/thejerf/suture/v4"
)

var DefaulDaemonConfig = config.AkaiDaemonConfig{}

type Daemon struct {
	config config.AkaiDaemonConfig
	sup    *suture.Supervisor // to coordinate all the services

	netScrapper NetworkScrapper
	dhtHost     DHTHost
	db          db.Database
	dataSampler *DataSampler

	network config.Network
}

// TODO: remove the API from the Daemon, and add it perhaps to the Network specifics
func NewDaemon(
	cfg config.AkaiDaemonConfig,
	h DHTHost,
	dbServ db.Database,
	dataSampler *DataSampler,
	netScrapper NetworkScrapper,
) (*Daemon, error) {
	daemon := &Daemon{
		config:      cfg,
		netScrapper: netScrapper,
		dhtHost:     h,
		db:          dbServ,
		sup:         suture.NewSimple("akai-daemon"),
		network:     config.NetworkFromStr(cfg.Network),
		dataSampler: dataSampler,
	}
	return daemon, nil
}

func (d *Daemon) Start(ctx context.Context) error {
	// prepare services
	err := d.db.Init(ctx, d.db.GetAllTables())
	if err != nil {
		return errors.Wrap(err, "unable to start akai-daemon")
	}

	if err := d.syncNetworkScrapper(ctx); err != nil {
		return err
	}

	// add services to
	d.sup.Add(d.netScrapper)
	d.sup.Add(d.db)
	d.sup.Add(d.dataSampler)

	return d.sup.Serve(ctx)
}

func (d *Daemon) syncNetworkScrapper(ctx context.Context) error {

	t := time.Now()
	firstItems, err := d.netScrapper.SyncWithDatabase(ctx)
	if err != nil {
		return err
	}
	if len(firstItems) > 0 {
		for _, item := range firstItems {
			_, validNextVisit := d.dataSampler.updateNextVisitTime(item)
			if !validNextVisit {
				continue
			}
			d.dataSampler.itemSet.addItem(item)
		}

		log.WithFields(log.Fields{
			"from key":    firstItems[0].Key,
			"to key":      firstItems[len(firstItems)-1].Key,
			"total items": len(firstItems),
			"import-time": time.Since(t),
		}).Info("synced data-sampler's sampleable items with the DB")
	} else {
		log.Info("no items found in db...")
	}
	return nil
}
