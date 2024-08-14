package amino

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ipfs/go-cid"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"

	"github.com/libp2p/go-libp2p"
	mplex "github.com/libp2p/go-libp2p-mplex"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	routingdisc "github.com/libp2p/go-libp2p/p2p/discovery/routing"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	record "github.com/libp2p/go-libp2p-record"

	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

type DHTHostConfig struct {
	HostID               int
	IP                   string
	Port                 int64
	DialTimeout          time.Duration
	DHTMode              kaddht.ModeOpt
	UserAgent            string
	V1Protocol           protocol.ID
	Bootstrapers         []peer.AddrInfo
	CustomValidator      record.Validator
	CustomProtocolPrefix *string
}

type DHTHost struct {
	cfg DHTHostConfig

	id      int
	host    host.Host
	dhtCli  *kaddht.IpfsDHT
	dhtDisc *routingdisc.RoutingDiscovery
}

func NewDHTHost(ctx context.Context, opts DHTHostConfig) (*DHTHost, error) {
	// prevent dial backoffs
	ctx = network.WithForceDirectDial(ctx, "prevent backoff")

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
		libp2p.Security(noise.ID, noise.New),
		libp2p.UserAgent(opts.UserAgent),
		libp2p.ResourceManager(rm),
		libp2p.DisableRelay(),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.Muxer(mplex.ID, mplex.DefaultTransport),
		libp2p.Muxer(yamux.ID, yamux.DefaultTransport),
	)
	if err != nil {
		return nil, err
	}
	err = bootstrapLibp2pHost(ctx, opts.HostID, h, opts.Bootstrapers)
	if err != nil {
		return nil, err
	}

	// DHT routing
	dhtOpts := []kaddht.Option{
		kaddht.Mode(opts.DHTMode),
		kaddht.BootstrapPeers(opts.Bootstrapers...),
		kaddht.V1ProtocolOverride(opts.V1Protocol),
	}
	// is there any need for a custom key-value validator?
	if opts.CustomValidator != nil {
		dhtOpts = append(dhtOpts, kaddht.Validator(opts.CustomValidator))
	}

	// is there a custom protocol-prefix?
	if opts.CustomProtocolPrefix != nil {
		dhtOpts = append(dhtOpts, kaddht.ProtocolPrefix(protocol.ID(*opts.CustomProtocolPrefix)))

	}
	dhtCli, err = kaddht.New(ctx, h, dhtOpts...)
	if err != nil {
		return nil, err
	}
	err = bootstrapDHT(ctx, opts.HostID, dhtCli)
	if err != nil {
		return nil, err
	}

	disc := routingdisc.NewRoutingDiscovery(dhtCli)

	// compose the DHT Host
	dhtHost := &DHTHost{
		cfg:     opts,
		id:      opts.HostID,
		host:    h,
		dhtCli:  dhtCli,
		dhtDisc: disc,
	}

	log.WithFields(log.Fields{
		"id":            opts.HostID,
		"agent_version": opts.UserAgent,
		"peer_id":       h.ID().String(),
		"multiaddrs":    h.Addrs(),
		"protocols":     h.Mux().Protocols(),
		"kad_dht":       opts.V1Protocol,
	}).Info("generated new amino dht host")

	return dhtHost, nil
}

func bootstrapLibp2pHost(ctx context.Context, id int, h host.Host, bootstrapers []peer.AddrInfo) error {
	var succCon int64
	var wg sync.WaitGroup

	hlog := log.WithField("host-id", id)

	for _, bnode := range bootstrapers {
		wg.Add(1)
		go func(bn peer.AddrInfo) {
			defer wg.Done()
			err := h.Connect(ctx, bn)
			if err != nil {
				hlog.Warnf("unable to connect bootstrap node: %s - %s", bn.String(), err.Error())
			} else {
				hlog.Debug("successful connection to bootstrap node:", bn.String())
				atomic.AddInt64(&succCon, 1)
			}
		}(bnode)
	}

	wg.Wait()

	// // debug the protocols by the bootstrappers
	// prots, err := h.Peerstore().GetProtocols(bootstrapers[0].ID)
	// if err == nil {
	// 	for _, prot := range prots {
	// 		fmt.Println(prot)
	// 	}
	// }

	// check connectivity with bootstrap nodes
	if succCon > 0 {
		hlog.Infof("host got connected to %d bootstrap nodes", succCon)
	} else {
		hlog.Warnf("unable to connect any of the bootstrap nodes from KDHT")
	}

	return nil
}

func bootstrapDHT(ctx context.Context, id int, dhtCli *kaddht.IpfsDHT) error {
	hlog := log.WithField("host-id", id)

	// bootstrap from existing connections
	err := dhtCli.Bootstrap(ctx)

	// force waiting a little bit to let the bootstrap work
	bootstrapTicker := time.NewTicker(5 * time.Second)
	select {
	case <-bootstrapTicker.C:
	case <-ctx.Done():
	}

	routingSize := dhtCli.RoutingTable().Size()
	if err != nil {
		hlog.Warnf("unable to bootstrap the dht-node %s", err.Error())
	}
	if routingSize == 0 {
		hlog.Warn("no error, but empty routing table after bootstraping")
	}
	log.WithField(
		"peers_in_routing", routingSize,
	).Info("dht cli bootstrapped")
	return nil
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
func (h *DHTHost) isPeerConnected(pID peer.ID) bool {
	// check if we already have a connection open to the peer
	peerList := h.host.Network().Peers()
	for _, p := range peerList {
		if p == pID {
			return true
		}
	}
	return false
}

func (h *DHTHost) FindClosestPeers(ctx context.Context, key string) (time.Duration, []peer.ID, error) {
	log.WithFields(log.Fields{
		"host-id": h.id,
		"cid":     key,
	}).Debug("looking for peers close to key")
	startT := time.Now()
	closePeers, err := h.dhtCli.GetClosestPeers(ctx, key)
	if err != nil {
		return time.Since(startT), []peer.ID{}, err
	}
	return time.Since(startT), closePeers, err
}

func (h *DHTHost) FindProviders(ctx context.Context, key cid.Cid) (time.Duration, []peer.AddrInfo, error) {
	log.WithFields(log.Fields{
		"host-id": h.id,
		"cid":     key.Hash().B58String(),
	}).Debug("looking for providers")
	startT := time.Now()
	providers, err := h.dhtCli.FindProviders(ctx, key)
	return time.Since(startT), providers, err
}

func (h *DHTHost) FindValue(
	ctx context.Context,
	key string) (time.Duration, []byte, error) {

	log.WithFields(log.Fields{
		"host-id": h.id,
		"key":     key,
	}).Info("looking for providers")

	startT := time.Now()
	providers, err := h.dhtCli.GetValue(
		ctx,
		key,
		kaddht.Quorum(1),
	)
	return time.Since(startT), providers, err
}

func (h *DHTHost) PutValue(
	ctx context.Context,
	key string,
	value []byte) (time.Duration, error) {

	log.WithFields(log.Fields{
		"host-id": h.id,
		"key":     key,
	}).Debug("looking for providers")
	startT := time.Now()
	err := h.dhtCli.PutValue(ctx, key, value)
	return time.Since(startT), err
}
