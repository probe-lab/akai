package host

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ipfs/go-cid"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"

	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

type DHTHostOpts struct {
	HostID       int
	IP           string
	Port         int
	DialTimeout  time.Duration
	UserAgent    string
	V1Protocol   protocol.ID
	Bootstrapers []peer.AddrInfo
}

type DHTHost struct {
	ctx context.Context

	cfg    DHTHostOpts
	id     int
	dhtCli *kaddht.IpfsDHT
	host   host.Host
}

func NewDHTHost(ctx context.Context, opts DHTHostOpts) (*DHTHost, error) {
	// prevent dial backoffs
	ctx = network.WithForceDirectDial(ctx, "prevent backoff")

	// Libp2p host configuration
	privKey, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate priv key for client's host")
	}
	// transport protocols
	mAddrs := make([]ma.Multiaddr, 0, 2)
	tcpAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", opts.IP, opts.Port))
	if err != nil {
		return nil, err
	}
	quicAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/%d/quic", opts.IP, opts.Port))
	if err != nil {
		return nil, err
	}
	mAddrs = append(mAddrs, tcpAddr, quicAddr)

	// resource manager
	limiter := rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits)
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		return nil, fmt.Errorf("new resource manager: %w", err)
	}

	// generate the libp2p host
	var dhtCli *kaddht.IpfsDHT
	h, err := libp2p.New(
		libp2p.WithDialTimeout(opts.DialTimeout),
		libp2p.ListenAddrs(mAddrs...),
		libp2p.Identity(privKey),
		libp2p.UserAgent(opts.UserAgent),
		libp2p.ResourceManager(rm),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			dhtOpts := make([]kaddht.Option, 0)
			dhtOpts = append(dhtOpts,
				kaddht.Mode(kaddht.ModeClient),
				kaddht.V1ProtocolOverride(opts.V1Protocol),
				kaddht.BootstrapPeers(opts.Bootstrapers...),
			)
			dhtCli, err = kaddht.New(ctx, h, dhtOpts...)
			return dhtCli, err
		}),
	)
	if err != nil {
		return nil, err
	}

	dhtHost := &DHTHost{
		ctx:    ctx,
		cfg:    opts,
		id:     opts.HostID,
		host:   h,
		dhtCli: dhtCli,
	}

	err = dhtHost.Init()
	if err != nil {
		return nil, errors.Wrap(err, "unable to init host")
	}

	log.WithFields(log.Fields{
		"id":         opts.HostID,
		"peer_id":    h.ID().String(),
		"multiaddrs": h.Addrs(),
	}).Info("generated new host")

	return dhtHost, nil
}

// Init makes sure that all the components of the DHT host are successfully initialized
func (h *DHTHost) Init() error {
	return h.bootstrap()
}

func (h *DHTHost) bootstrap() error {
	var succCon int64
	var wg sync.WaitGroup

	hlog := log.WithField("host-id", h.id)
	for _, bnode := range h.cfg.Bootstrapers {
		wg.Add(1)
		go func(bn peer.AddrInfo) {
			defer wg.Done()
			err := h.host.Connect(h.ctx, bn)
			if err != nil {
				hlog.Debug("unable to connect bootstrap node:", bn.String())
			} else {
				hlog.Debug("successful connection to bootstrap node:", bn.String())
				atomic.AddInt64(&succCon, 1)
			}
		}(bnode)
	}

	wg.Wait()

	var err error
	if succCon > 0 {
		hlog.Infof("host got connected to %d bootstrap nodes", succCon)
	} else {
		err = errors.New("unable to connect any of the bootstrap nodes from KDHT")
	}
	return err
}

func (h *DHTHost) IntenalID() int {
	return h.id
}

func (h *DHTHost) ID() peer.ID {
	return h.host.ID()
}

func (h *DHTHost) Host() host.Host {
	return h.host
}

func (h *DHTHost) GetMAddrsOfPeer(p peer.ID) []ma.Multiaddr {
	return h.host.Peerstore().Addrs(p)
}

// conside moving this to the Host
func (h *DHTHost) isPeerConnected(pId peer.ID) bool {
	// check if we already have a connection open to the peer
	peerList := h.host.Network().Peers()
	for _, p := range peerList {
		if p == pId {
			return true
		}
	}
	return false
}

func (h *DHTHost) FindProviders(
	ctx context.Context,
	key cid.Cid) (time.Duration, []peer.AddrInfo, error) {

	log.WithFields(log.Fields{
		"host-id": h.id,
		"cid":     key.Hash().B58String(),
	}).Debug("looking for providers")
	startT := time.Now()
	providers, err := h.dhtCli.FindProviders(ctx, key)
	return time.Since(startT), providers, err
}
