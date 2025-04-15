package core

import (
	"context"

	"github.com/probe-lab/akai/avail"
	"github.com/probe-lab/akai/db/models"
)

// NetworkScrapper is the main interface that each of the network should follow
// in order to provide new items to the daemon for periodic sampling
type NetworkScrapper interface {
	GetSamplingItemStream(context.Context) (chan []*models.SamplingItem, error)
	SyncWithDatabase(context.Context) ([]*models.SamplingItem, error)
	Serve(context.Context) error
	Close(context.Context) error
}

// check that the AvailNetworkScrapper is compatible with the Core interface
var _ NetworkScrapper = (*avail.NetworkScrapper)(nil)
