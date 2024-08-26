package core

import (
	"context"
	"time"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/db/models"
)

var DefaultDataSamplerConfig = config.AkaiDataSamplerConfig{
	Network:         config.DefaultNetwork.String(),
	Workers:         10,
	SamplingTimeout: 10 * time.Second,
	DBsyncInterval:  10 * time.Minute,
}

type DataSampler struct {
	cfg config.AkaiDataSamplerConfig

	database db.Database
}

func NewDataSampler(
	cfg config.AkaiDataSamplerConfig,
	database db.Database) (*DataSampler, error) {

	ds := &DataSampler{
		cfg:      cfg,
		database: database,
	}
	return ds, nil
}

func (ds *DataSampler) Serve(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (ds *DataSampler) Init(ctx context.Context) error {

	return nil
}

func (ds *DataSampler) syncWithDatabase(ctx context.Context) error {

	return nil
}

func (ds *DataSampler) sampleSegment(ctx context.Context, segment models.AgnosticSegment) error {

	return nil
}
