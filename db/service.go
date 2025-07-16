package db

import (
	"context"
	"fmt"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/clickhouse"
	"github.com/probe-lab/akai/db/models"
)

var DefaultConnectionDetails = &config.DatabaseDetails{
	Driver:      "clickhouse",
	Address:     "127.0.0.1:9000",
	User:        "username",
	Password:    "password",
	Database:    "akai_test",
	TLSrequired: false,
}

type Database interface {
	Init(context.Context, map[string]struct{}) error
	Serve(context.Context) error
	Close(context.Context) error
	// tables's perspective
	GetAllTables() map[string]struct{}
	// blocks
	PersistNewBlock(context.Context, *models.Block) error
	GetSampleableBlocks(context.Context) ([]models.Block, error)
	GetAllBlocks(context.Context) ([]models.Block, error)
	GetLastBlock(context.Context) (models.Block, error)
	// sample items
	PersistNewSamplingItem(context.Context, *models.SamplingItem) error
	PersistNewSamplingItems(context.Context, []*models.SamplingItem) error
	GetSampleableItems(context.Context) ([]models.SamplingItem, error)
	// cell visists
	PersistNewSampleGenericVisit(context.Context, *models.SampleGenericVisit) error
	PersistNewSampleValueVisit(context.Context, *models.SampleValueVisit) error
	PersistNewPeerInfoVisit(context.Context, *models.PeerInfoVisit) error
}

var _ Database = (*clickhouse.ClickHouseDB)(nil)

func NewDatabase(details *config.DatabaseDetails, networkConfig *config.NetworkConfiguration) (Database, error) {
	switch details.Driver {
	case "clickhouse-local":
		return clickhouse.NewClickHouseDB(details, clickhouse.ClickhouseLocalInstance, networkConfig.Network)
	case "clickhouse-replicated":
		return clickhouse.NewClickHouseDB(details, clickhouse.ClickhouseReplicatedInstance, networkConfig.Network)
	default:
		return nil, fmt.Errorf("not recognized database diver (%s)", details.Driver)
	}
}
