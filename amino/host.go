package amino

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/probe-lab/akai/config"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"

	"github.com/libp2p/go-libp2p"
	mplex "github.com/libp2p/go-libp2p-mplex"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	routingdisc "github.com/libp2p/go-libp2p/p2p/discovery/routing"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	record "github.com/libp2p/go-libp2p-record"

	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

const summaryFreq = 30 * time.Second

type DHTHostConfig struct {
	HostID               int
	IP                   string
	Port                 int64
	DialTimeout          time.Duration
	DHTMode              kaddht.ModeOpt
	AgentVersion         string
	V1Protocol           protocol.ID
	Bootstrapers         []peer.AddrInfo
	CustomValidator      record.Validator
	CustomProtocolPrefix *string
	Meter                metric.Meter
}

type DHTHost struct {
	cfg    *DHTHostConfig
	netCfg *config.NetworkConfiguration

	id      int
	host    host.Host
	dhtCli  *kaddht.IpfsDHT
	dhtDisc *routingdisc.RoutingDiscovery

	// Metrics
	connCount        metric.Int64ObservableGauge
	routingPeerCount metric.Int64ObservableGauge
}

func NewDHTHost(ctx context.Context, opts *DHTHostConfig, netCfg *config.NetworkConfiguration) (*DHTHost, error) {
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
		libp2p.UserAgent(opts.AgentVersion),
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

	// DHT routing
	dhtOpts := []kaddht.Option{
		kaddht.Mode(opts.DHTMode),
		kaddht.BootstrapPeers(opts.Bootstrapers...),
	}
	// is there any need for a custom key-value validator?
	if opts.CustomValidator != nil {
		dhtOpts = append(dhtOpts, kaddht.Validator(opts.CustomValidator))
	}

	// is there a custom protocol-prefix?
	if opts.CustomProtocolPrefix != nil {
		dhtOpts = append(dhtOpts, kaddht.ProtocolPrefix(protocol.ID(*opts.CustomProtocolPrefix)))
	}

	// // overrido custom V1 protocol
	if opts.V1Protocol != "" {
		dhtOpts = append(dhtOpts, kaddht.V1ProtocolOverride(opts.V1Protocol))
	}

	dhtCli, err = kaddht.New(ctx, h, dhtOpts...)
	if err != nil {
		return nil, err
	}

	err = bootstrapDHT(ctx, opts.HostID, dhtCli, opts.Bootstrapers)
	if err != nil {
		return nil, err
	}

	disc := routingdisc.NewRoutingDiscovery(dhtCli)

	// compose the DHT Host
	dhtHost := &DHTHost{
		cfg:     opts,
		netCfg:  netCfg,
		id:      opts.HostID,
		host:    h,
		dhtCli:  dhtCli,
		dhtDisc: disc,
	}

	// debug bootnodes
	for _, bootnode := range opts.Bootstrapers {
		attrs, err := dhtHost.getLibp2pHostInfo(bootnode.ID)
		if err != nil {
			log.Warnf("loading host (%s) info: %s", bootnode.ID.String(), err.Error())
		} else {
			log.WithFields(log.Fields{
				"agent_version":     attrs["agent_version"],
				"protocols":         attrs["protocols"],
				"protocol_versions": attrs["protocol_versions"],
			}).Info("bootnode info")
		}
	}

	err = dhtHost.initMetrics()
	if err != nil {
		return nil, err
	}

	go dhtHost.internalsDebugger(ctx)

	log.WithFields(log.Fields{
		"id":            opts.HostID,
		"agent_version": opts.AgentVersion,
		"peer_id":       h.ID().String(),
		"multiaddrs":    h.Addrs(),
		"protocols":     h.Mux().Protocols(),
		"kad_dht":       opts.V1Protocol,
	}).Info("generated new amino dht host")

	return dhtHost, nil
}

func bootstrapDHT(ctx context.Context, id int, dhtCli *kaddht.IpfsDHT, bootstrappers []peer.AddrInfo) error {
	hlog := log.WithField("host-id", id)

	// connect to the bootnodes
	var wg sync.WaitGroup

	succBootnodes := 0
	for _, bnode := range bootstrappers {
		wg.Add(1)
		go func(bn peer.AddrInfo) {
			defer wg.Done()
			err := dhtCli.Host().Connect(ctx, bn)
			if err != nil {
				hlog.Warnf("unable to connect bootstrap node: %s - %s", bn.String(), err.Error())
			} else {
				succBootnodes++
				hlog.Debug("successful connection to bootstrap node:", bn.String())
			}
		}(bnode)
	}

	// bootstrap from existing connections
	wg.Wait()
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
	log.WithFields(log.Fields{
		"successful-bootnodes": succBootnodes,
		"peers_in_routing":     routingSize,
	}).Info("dht cli bootstrapped")
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
	key string,
) (t time.Duration, value []byte, err error) {
	log.WithFields(log.Fields{
		"host-id": h.id,
		"key":     key,
	}).Debug("looking for providers")

	opCtx, opCancel := context.WithCancel(ctx)
	defer opCancel()
	startT := time.Now()
	outC, err := h.dhtCli.SearchValue(
		opCtx,
		key,
		kaddht.Quorum(1),
	)

	var ok bool
	select {
	case value, ok = <-outC:
		// pass (record value)
	case <-opCtx.Done():
		// pass (deadline exceeded)
	}
	if len(value) <= 0 && err == nil {
		if !ok {
			err = routing.ErrNotFound
		} else {
			err = context.DeadlineExceeded
		}
	}
	return time.Since(startT), value, err
}

func (h *DHTHost) PutValue(ctx context.Context, key string, value []byte) (time.Duration, error) {
	log.WithFields(log.Fields{
		"host-id": h.id,
		"key":     key,
	}).Debug("looking for providers")
	startT := time.Now()
	err := h.dhtCli.PutValue(ctx, key, value)

	return time.Since(startT), err
}

func (h *DHTHost) FindPeers(ctx context.Context, key string, opDuration time.Duration) (time.Duration, []peer.AddrInfo, error) {
	log.WithFields(log.Fields{
		"host-id": h.id,
		"key":     key,
	}).Info("looking for providers")

	providers := make(map[peer.ID]peer.AddrInfo)
	res := make([]peer.AddrInfo, 0)
	timeoutT := time.NewTicker(opDuration)

	startT := time.Now()
	peersC, err := h.dhtDisc.FindPeers(ctx, key, discovery.Limit(0))
	if err != nil {
		return time.Since(startT), res, err
	}

waitLoop:
	for {
		select {
		case newPeer, ok := <-peersC:
			if !ok {
				break waitLoop
			}

			_, ok = providers[newPeer.ID]
			if !ok || (ok && len(newPeer.Addrs) > 0) {
				providers[newPeer.ID] = newPeer
			}
		case <-timeoutT.C:
			break waitLoop
		}
	}

	for _, val := range providers {
		res = append(res, val)
	}

	return time.Since(startT), res, err
}

// initMetrics initializes various prometheus metrics and stores the meters
// on the [Node] object.
func (h *DHTHost) initMetrics() (err error) {
	h.connCount, err = h.cfg.Meter.Int64ObservableGauge("current_connections")
	if err != nil {
		return fmt.Errorf("new current_connections counter: %w", err)
	}

	_, err = h.cfg.Meter.RegisterCallback(func(ctx context.Context, obs metric.Observer) error {
		obs.ObserveInt64(h.connCount, int64(len(h.host.Network().Peers())))
		return nil
	}, h.connCount)
	if err != nil {
		return fmt.Errorf("register current_connections counter callback: %w", err)
	}

	h.routingPeerCount, err = h.cfg.Meter.Int64ObservableGauge("routing_peers")
	if err != nil {
		return fmt.Errorf("new routing_peers counter: %w", err)
	}

	_, err = h.cfg.Meter.RegisterCallback(func(ctx context.Context, obs metric.Observer) error {
		obs.ObserveInt64(h.routingPeerCount, int64(h.dhtCli.RoutingTable().Size()))
		return nil
	}, h.routingPeerCount)
	if err != nil {
		return fmt.Errorf("register routing_peers counter callback: %w", err)
	}
	return nil
}

func (h *DHTHost) internalsDebugger(ctx context.Context) {
	tick := time.NewTicker(summaryFreq)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			peers := h.getCurrentConnections()
			// debug bootnodes
			log.WithFields(log.Fields{
				"peer-connections": len(peers),
				"routing-nodes":    h.dhtCli.RoutingTable().Size(),
			}).Info("connectivity summary:")
			for idx, peer := range peers {
				attrs, err := h.getLibp2pHostInfo(peer)
				if err != nil {
					continue
				}
				log.WithFields(log.Fields{
					"agent_version":     attrs["agent_version"],
					"protocols":         attrs["protocols"],
					"protocol_versions": attrs["protocol_versions"],
				}).Debugf("	* peer (%d): %s", idx, peer.String())
			}
			tick.Reset(summaryFreq)
		}
	}
}

func (h *DHTHost) getCurrentConnections() []peer.ID {
	return h.host.Network().Peers()
}

func (h *DHTHost) getLibp2pHostInfo(pID peer.ID) (map[string]any, error) {
	attrs := make(map[string]any)
	// read from the local peerstore
	// agent version
	var av any = "unknown"
	av, err := h.host.Peerstore().Get(pID, "AgentVersion")
	if err != nil {
		return attrs, err
	}
	attrs["agent_version"] = av

	// protocols
	prots, err := h.host.Network().Peerstore().GetProtocols(pID)
	if err != nil {
		return attrs, err
	}
	attrs["protocols"] = prots

	// protocol version
	var pv any = "unknown"
	pv, err = h.host.Peerstore().Get(pID, "ProtocolVersion")
	if err != nil {
		return attrs, err
	}
	attrs["protocol_version"] = pv

	return attrs, nil
}
