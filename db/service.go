package db

import (
	"context"
	"time"

	"github.com/probe-lab/akai/models"
)

var (
	MaxFlushInterval = 5 * time.Second
)

type Database interface {
	Init(context.Context, map[string]struct{}) error
	Close() error
	// tables's perspective
	GetAllTables() map[string]struct{}
	// networks
	GetNetworks(context.Context) ([]Network, error)
	// blocks
	PersistNewBlock(context.Context) error
	GetSampleableBlocks(context.Context) ([]models.AgnosticBlock, error)
	// cell visists
	PersistNewCellVisit(context.Context) error
}
