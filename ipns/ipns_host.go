package amino

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ipfs/boxo/ipns"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/probe-lab/akai/core"
	"github.com/sirupsen/logrus"
)

type IpnsConfig struct {
	Timeout time.Duration
	Quorum  int
}

type IpnsClient struct {
	config IpnsConfig
	rt     core.DHTHost
}

func NewIpnsClient(cfg IpnsConfig, dhtCli core.DHTHost) (*IpnsClient, error) {
	return &IpnsClient{
		config: cfg,
		rt:     dhtCli,
	}, nil
}

type IpnsValueResult struct {
	OpDuration time.Duration
	Type       string
	Value      string
	TTL        time.Duration
	IsValid    bool
	Error      error
}

func (c *IpnsClient) ResolveIPNS(ctx context.Context, name string, keyType string) (*IpnsValueResult, error) {
	logrus.WithFields(logrus.Fields{
		"name": name,
		"type": keyType,
	}).Info("getting ipns record")

	switch keyType {
	case "cid":
		key, err := ipns.NameFromString(name)
		if err != nil {
			return nil, err
		}
		/*
			dhtKey, err := ComposeIpnsKey(name)
			if err != nil {
				return nil, fmt.Errorf("unable to parse ipns-key to name - %s", err.Error())
			}
		*/
		return c.resolveDHT(ctx, string(key.RoutingKey()))
	case "dns":
		return c.resolveDNS(ctx, name)
	default:
		return nil, fmt.Errorf("not recognized type %s", keyType)
	}
}

func (c *IpnsClient) resolveDHT(ctx context.Context, name string) (*IpnsValueResult, error) {
	logrus.WithFields(logrus.Fields{
		"name": name,
	}).Info("getting ipns record from the dht")

	// compose results
	var values [][]byte
	res := &IpnsValueResult{
		Type:  "raw-string",
		Error: nil,
	}

	res.OpDuration, values, res.Error = c.rt.FindQuorumValue(ctx, name, c.config.Timeout, c.config.Quorum)
	if res.Error != nil {
		return res, res.Error
	}

	currentSeq := uint64(0)
	for i, bytes := range values {
		var isValid = false
		var path path.Path
		var nextSeq = uint64(0)
		var ttl time.Duration
		defer func() {
			if (path.String() != res.Value && nextSeq > currentSeq) || i == 0 {
				currentSeq = nextSeq
				res.Value = path.String()
				res.TTL = ttl
				res.IsValid = isValid
				res.Error = nil
			}
		}()
		// parse the bytes into the IPNS-record structure
		rec, err := ipns.UnmarshalRecord(bytes)
		if err != nil {
			logrus.Errorf("unmarshaling the record from the recv bytes: %s", err.Error())
			continue
		}

		var pubkey crypto.PubKey
		pubkey, err = rec.PubKey()
		if err != nil {
			logrus.Warnf("retrieving the pubkey from the record: %s", err.Error())
		} else {
			// validate the record
			err = ipns.Validate(rec, pubkey)
			if err != nil {
				logrus.Warnf("validating the signature of the record: %s", err.Error())
				continue
			} else {
				isValid = true
			}
		}

		// get the value
		path, err = rec.Value()
		if err != nil {
			logrus.Errorf("unable to get cid from record: %s", err.Error())
			continue
		}

		// get the TTL
		ttl, err = rec.TTL()
		if err != nil {
			logrus.Warnf("extracting the sequence from the record: %s", err.Error())
		}

		// check seq number
		nextSeq, err = rec.Sequence()
		if err != nil {
			logrus.Warnf("extracting the sequence from the record: %s", err.Error())
		}
	}
	return res, nil
}

func (c *IpnsClient) resolveDNS(ctx context.Context, k string) (*IpnsValueResult, error) {
	res := &IpnsValueResult{
		Type:  "dns",
		Error: nil,
	}

	startT := time.Now()
	path, err := ResolveDNSLink(ctx, k)
	res.OpDuration = time.Since(startT)
	if err != nil {
		return res, err
	}
	res.Value = path
	return res, nil
}

func ComposeIpnsKey(k string) (string, error) {
	peerIDstr := strings.Trim(k, "/ipns/")
	c, err := cid.Decode(peerIDstr)
	if err != nil {
		return "", err
	}
	pid, err := peer.FromCid(c)
	if err != nil {
		return "", err
	}
	st := ipns.NameFromPeer(pid)
	s := string(st.RoutingKey())
	return s, nil
}

func ResolveDNSLink(ctx context.Context, domain string) (string, error) {
	dnslinkDomain := "_dnslink." + domain

	txts, err := net.DefaultResolver.LookupTXT(ctx, dnslinkDomain)
	if err != nil {
		return "", fmt.Errorf("failed to lookup TXT records: %w", err)
	}

	for _, txt := range txts {
		if strings.HasPrefix(txt, "dnslink=") {
			return strings.TrimPrefix(txt, "dnslink="), nil
		}
	}

	return "", fmt.Errorf("no dnslink record found in TXT records")
}
