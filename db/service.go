package db

import (
	"context"

	"github.com/probe-lab/akai/models"
)

type Database interface {
	Init(context.Context) error
	Run(context.Context) error
	Stop() error
	// tables's perspective
	// networks
	EnsureNetworks()
	GetNetworks() ([]Network, error)
	// blocks
	PersistNewBlock() error
	GetSampleableBlocks() ([]models.AgnosticBlock, error)
	// cell visists
	PersistNewCellVisit() error
}
