package core

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
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
	"github.com/sirupsen/logrus"
)

// TODO: in the future, this could be extended to make a "smarter" division of the tasks:
// - key-space filltering
// - host selection
// - or similars
type SamplingTask struct {
	VisitRound int
	Item       *models.SamplingItem
	Quorum     int
}

type SamplerFunction func(
	context.Context,
	DHTHost,
	SamplingTask,
	time.Duration) (models.GeneralVisit, error)

// logical sampler functions
func SampleByFindProviders(
	ctx context.Context,
	h DHTHost,
	task SamplingTask,
	timeout time.Duration,
) (models.GeneralVisit, error) {
	// generate the base for the generic visit
	visit := models.SampleGenericVisit{
		VisitType:     config.SampleProviders.String(),
		VisitRound:    uint64(task.VisitRound),
		Network:       task.Item.Network,
		Timestamp:     time.Now(),
		Key:           task.Item.Key,
		DurationMs:    int64(0),
		ResponseItems: 0,
		Peers:         make([]string, 0),
		Error:         "",
	}

	contentID, err := cid.Decode(task.Item.Key)
	if err != nil {
		visit.Error = "Item key doesn't represent a valid CID"
	}

	// make the sampling
	duration, providers, err := h.FindProviders(ctx, contentID, timeout)
	if err != nil {
		visit.Error = err.Error()
	}
	// apply rest of values
	visit.DurationMs = duration.Milliseconds()
	visit.ResponseItems = int32(len(providers))
	for _, pr := range providers {
		visit.Peers = append(visit.Peers, pr.ID.String())
	}

	logrus.WithFields(logrus.Fields{
		"timestamp":   time.Now(),
		"key":         task.Item.Key,
		"duration":    duration,
		"n_providers": len(providers),
		"providers":   len(visit.Peers),
		"error":       visit.Error,
	}).Debug("find providers operation done")

	return models.GeneralVisit{
		GenericVisit: []*models.SampleGenericVisit{
			&visit,
		},
		GenericValueVisit: nil,
	}, nil
}

func SampleByFindPeers(
	ctx context.Context,
	h DHTHost,
	task SamplingTask,
	timeout time.Duration,
) (models.GeneralVisit, error) {
	visit := models.SampleGenericVisit{
		VisitType:     config.SamplePeers.String(),
		VisitRound:    uint64(task.VisitRound),
		Network:       task.Item.Network,
		Timestamp:     time.Now(),
		Key:           task.Item.Key,
		DurationMs:    int64(0),
		ResponseItems: 0,
		Peers:         make([]string, 0),
		Error:         "",
	}

	// make the sampling
	duration, peers, err := h.FindPeers(ctx, task.Item.Key, timeout)
	if err != nil {
		visit.Error = err.Error()
	}

	// apply rest of values
	visit.DurationMs = duration.Milliseconds()
	visit.ResponseItems = int32(len(peers))
	for _, pr := range peers {
		visit.Peers = append(visit.Peers, pr.ID.String())
	}

	logrus.WithFields(logrus.Fields{
		"timestamp":   time.Now(),
		"key":         task.Item.Key,
		"duration":    duration,
		"n_providers": len(peers),
		"peer_ids":    ListPeerIDsFromAddrsInfos(peers),
		"error":       visit.Error,
	}).Debug("find peers operation done")

	return models.GeneralVisit{
		GenericVisit: []*models.SampleGenericVisit{
			&visit,
		},
		GenericValueVisit: nil,
	}, nil
}

func SampleByFindPeerInfo(
	ctx context.Context,
	h DHTHost,
	task SamplingTask,
	timeout time.Duration,
) (models.GeneralVisit, error) {
	visit := models.PeerInfoVisit{
		VisitRound:      uint64(task.VisitRound),
		Timestamp:       time.Now(),
		Network:         task.Item.Network,
		PeerID:          task.Item.Key,
		Duration:        0,
		AgentVersion:    "",
		Protocols:       make([]protocol.ID, 0),
		ProtocolVersion: "",
		MultiAddresses:  make([]multiaddr.Multiaddr, 0),
		Error:           "",
	}

	peerID, err := peer.Decode(task.Item.Key)
	if err != nil {
		return models.GeneralVisit{}, err
	}

	logrus.WithFields(logrus.Fields{"timeout": timeout})

	duration, peerInfo, err := h.FindPeer(ctx, peerID, timeout)
	visit.Duration = duration
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Warnf("find peer info operation could not be completed")
		visit.Error = err.Error()
		return models.GeneralVisit{
			GenericPeerInfoVisit: []*models.PeerInfoVisit{
				&visit,
			},
		}, err
	}

	hostInfo, err := h.ConnectAndIdentifyPeer(ctx, peerInfo, 1, 30*time.Second)
	errStr := ""
	if err != nil {
		visit.Error = err.Error()
	} else {
		visit.Error = errStr
	}

	logrus.WithFields(logrus.Fields{
		"operation":   config.SamplePeerInfo.String(),
		"timestamp":   duration.Milliseconds(),
		"peer_id":     peerInfo.ID.String(),
		"maddres":     peerInfo.Addrs,
		"peer_info":   hostInfo,
		"duration_ms": duration,
		"error":       errStr,
	}).Infof("find peer info operation done")

	// apply remaining values
	if agentVersion, ok := hostInfo["agent_version"].(string); ok {
		visit.AgentVersion = agentVersion
	}
	if protocolVersion, ok := hostInfo["protocol_version"].(string); ok {
		visit.ProtocolVersion = protocolVersion
	}
	visit.Protocols = append(visit.Protocols, hostInfo["protocols"].([]protocol.ID)...)
	visit.MultiAddresses = append(visit.MultiAddresses, peerInfo.Addrs...)

	return models.GeneralVisit{
		GenericPeerInfoVisit: []*models.PeerInfoVisit{
			&visit,
		},
	}, nil
}

func SampleByFindValue(
	ctx context.Context,
	h DHTHost,
	task SamplingTask,
	timeout time.Duration,
) (models.GeneralVisit, error) {
	visit := models.SampleValueVisit{
		VisitRound:    uint64(task.VisitRound),
		VisitType:     task.Item.ItemType,
		Network:       task.Item.Network,
		Timestamp:     time.Now(),
		Key:           task.Item.Key,
		BlockNumber:   task.Item.BlockLink,
		DASRow:        task.Item.DASRow,
		DASColumn:     task.Item.DASColumn,
		DurationMs:    int64(0),
		IsRetrievable: false,
		Bytes:         0,
		Error:         "",
	}

	// make the sampling
	duration, bytes, err := h.FindValue(ctx, task.Item.Key, timeout)
	if err != nil {
		visit.Error = err.Error()
	}
	if len(bytes) > 0 {
		visit.IsRetrievable = true
		visit.Bytes = int32(len(bytes))
		// there is an edgy case where the sampling reports a failure
		// but the content was retrieved -> Err = ContextDeadlineExceeded
		// rewrite the error to nil/Empty
		if err == context.DeadlineExceeded {
			logrus.WithFields(logrus.Fields{
				"key":   task.Item.Key,
				"bytes": len(bytes),
			}).Warn("key retrieved, but context was exceeded")
			visit.Error = ""
		}
	}
	// apply rest of values
	visit.DurationMs = duration.Milliseconds()

	logrus.WithFields(logrus.Fields{
		"timestamp": time.Now(),
		"key":       task.Item.Key,
		"duration":  duration,
		"bytes":     len(bytes),
		"error":     visit.Error,
	}).Debug("find value operation done")

	return models.GeneralVisit{
		GenericVisit: nil,
		GenericValueVisit: []*models.SampleValueVisit{
			&visit,
		},
	}, nil
}

func ResolveIPNSRecord(
	ctx context.Context,
	h DHTHost,
	task SamplingTask,
	timeout time.Duration,
) (models.GeneralVisit, error) {
	logrus.WithFields(logrus.Fields{
		"name": task.Item.Key,
	}).Info("getting ipns record from the dht")

	// compose results
	var values [][]byte
	res := &models.IPNSRecordVisit{
		VisitRound:    uint64(task.VisitRound),
		Timestamp:     time.Now(),
		Record:        task.Item.Key,
		RecordType:    task.Item.ItemType,
		Quorum:        uint8(task.Quorum),
		SeqNumber:     uint32(0),
		TTL:           time.Duration(0),
		IsValid:       false,
		IsRetrievable: false,
		Result:        "",
		Duration:      time.Duration(0),
		Error:         "",
	}
	gv := models.GeneralVisit{
		GenericIPNSRecordVisit: []*models.IPNSRecordVisit{
			res,
		},
	}

	rtKey, err := config.ComposeIpnsKey(task.Item.Key)
	if err != nil {
		res.Error = err.Error()
		return gv, nil
	}

	res.Duration, values, err = h.FindQuorumValue(ctx, string(rtKey), timeout, task.Quorum)
	if err != nil {
		res.Error = err.Error()
		return gv, nil
	}

	var resErr error
	var valErr error
	currentSeq := uint64(0)
	for i, bytes := range values {
		var isValid = false
		var nextSeq = uint64(0)
		var ttl time.Duration
		var rec *ipns.Record
		var path path.Path

		// parse the bytes into the IPNS-record structure
		rec, resErr = ipns.UnmarshalRecord(bytes)
		if err != nil {
			logrus.Errorf("unmarshaling the record from the recv bytes: %s", err.Error())
			continue
		}

		// get the value
		path, valErr = rec.Value()
		if err != nil {
			logrus.Errorf("unable to get cid from record: %s", err.Error())
			continue
		} else {
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

			// get the pubkey and validate the record
			var pubkey crypto.PubKey
			pubkey, err = rec.PubKey()
			if err != nil {
				logrus.Warnf("retrieving the pubkey from the record: %s", err.Error())
			} else {
				// validate the record
				err = ipns.Validate(rec, pubkey)
				if err != nil {
					logrus.Warnf("validating the signature of the record: %s", err.Error())
				} else {
					isValid = true
				}
			}
		}

		if (path.String() != "" && nextSeq >= currentSeq) || i == 0 {
			currentSeq = nextSeq
			res.SeqNumber = uint32(nextSeq)
			res.Result = path.String()
			res.TTL = ttl
			res.IsValid = isValid
			res.IsRetrievable = true
			res.Error = "" // reset the err
		}
	}
	if res.Record == "" && res.Error == "" {
		if resErr != nil {
			res.Error = resErr.Error()
		}
		if valErr != nil {
			res.Error = valErr.Error()
		}
	}
	return gv, nil
}

func ResolveDNS(
	ctx context.Context,
	_ DHTHost,
	task SamplingTask,
	timeout time.Duration,
) (models.GeneralVisit, error) {
	res := &models.IPNSRecordVisit{
		VisitRound:    uint64(task.VisitRound),
		Timestamp:     time.Now(),
		Record:        task.Item.Key,
		RecordType:    task.Item.ItemType,
		Quorum:        uint8(task.Quorum),
		SeqNumber:     uint32(0),
		TTL:           time.Duration(0),
		IsValid:       false,
		IsRetrievable: false,
		Result:        "",
		Duration:      time.Duration(0),
		Error:         "",
	}
	gv := models.GeneralVisit{
		GenericIPNSRecordVisit: []*models.IPNSRecordVisit{
			res,
		},
	}

	startT := time.Now()
	path, err := ResolveDNSLink(ctx, task.Item.Key, timeout)
	res.Duration = time.Since(startT)
	if err != nil {
		res.Error = err.Error()
		return gv, nil
	}
	res.Result = path
	res.IsValid = true
	res.IsRetrievable = true
	return gv, nil
}

func ResolveDNSLink(ctx context.Context, domain string, timeout time.Duration) (string, error) {
	opCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	dnslinkDomain := "_dnslink." + domain
	txts, err := net.DefaultResolver.LookupTXT(opCtx, dnslinkDomain)
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

func samplingFnFromType(sampleType config.SamplingType) (SamplerFunction, error) {
	switch sampleType {
	case config.SampleProviders:
		return SampleByFindProviders, nil

	case config.SampleValue:
		return SampleByFindValue, nil

	case config.SamplePeers:
		return SampleByFindPeers, nil

	case config.SamplePeerInfo:
		return SampleByFindPeerInfo, nil

	case config.SampleIPNSname:
		return ResolveIPNSRecord, nil

	case config.SampleDNSlink:
		return ResolveDNS, nil
	default:
		return func(
			_ context.Context,
			_ DHTHost,
			_ SamplingTask,
			_ time.Duration,
		) (models.GeneralVisit, error) {
			return models.GeneralVisit{}, nil
		}, nil
	}
}
