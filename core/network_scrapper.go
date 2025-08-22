package core

import (
	"context"

	"github.com/probe-lab/akai/avail"
	"github.com/probe-lab/akai/celestia"
	"github.com/probe-lab/akai/db/models"
	"github.com/probe-lab/akai/ipfs"
	"github.com/probe-lab/akai/ipns"
)

// NetworkScrapper is the main interface that each of the network should follow
// in order to provide new items to the daemon for periodic sampling
type NetworkScrapper interface {
	GetSamplingItemStream() chan []*models.SamplingItem
	SyncWithDatabase(context.Context) ([]*models.SamplingItem, error)
	GetQuorum() int
	Serve(context.Context) error
	Close(context.Context) error
}

// check that the AvailNetworkScrapper is compatible with the Core interface
var _ NetworkScrapper = (*avail.NetworkScrapper)(nil)
var _ NetworkScrapper = (*celestia.NetworkScrapper)(nil)
var _ NetworkScrapper = (*ipfs.NetworkScrapper)(nil)
var _ NetworkScrapper = (*ipns.NetworkScrapper)(nil)
