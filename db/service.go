package db

import (
	"context"

	"github.com/probe-lab/akai/models"
)

type Database interface {
	Init(context.Context, map[string]struct{}) error
	Run(context.Context) error
	Close() error
	// tables's perspective
	// networks
	GetNetworks(context.Context) ([]Network, error)
	// blocks
	PersistNewBlock(context.Context) error
	GetSampleableBlocks(context.Context) ([]models.AgnosticBlock, error)
	// cell visists
	PersistNewCellVisit(context.Context) error
}
