package db

import (
	"context"
	"fmt"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/clickhouse"
	"github.com/probe-lab/akai/db/models"
)

var DefaultConnectionDetails = config.DatabaseDetails{
	Driver:   "clickhouse",
	Address:  "127.0.0.1:9000",
	User:     "username",
	Password: "password",
	Database: "akai_test",
	Params:   "",
}

type Database interface {
	Init(context.Context, map[string]struct{}) error
	Serve(context.Context) error
	Close(context.Context) error
	// tables's perspective
	GetAllTables() map[string]struct{}
	// networks
	PersistNewNetwork(context.Context, models.Network) error
	GetNetworks(context.Context) ([]models.Network, error)
	// blocks
	PersistNewBlob(context.Context, models.AgnosticBlob) error
	GetSampleableBlobs(context.Context) ([]models.AgnosticBlob, error)
	GetAllBlobs(context.Context) ([]models.AgnosticBlob, error)
	// samples
	PersistNewSegment(context.Context, models.AgnosticSegment) error
	PersistNewSegments(context.Context, []models.AgnosticSegment) error
	GetSampleableSegments(context.Context) ([]models.AgnosticSegment, error)
	// cell visists
	PersistNewSegmentVisit(context.Context) error
}

var _ Database = (*clickhouse.ClickHouseDB)(nil)

func NewDatabase(details config.DatabaseDetails, network models.Network) (Database, error) {
	switch details.Driver {
	case "clickhouse":
		return clickhouse.NewClickHouseDB(details, network)
	default:
		return nil, fmt.Errorf("not recognized database diver (%s)", details.Driver)
	}
}
