package config

import (
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
)

type SamplingItemType uint8

func (i SamplingItemType) String() string {
	switch i {
	case UnknownItemType:
		return "UNKNOWN"
	case IPFSCidItemType:
		return "IPFS_CID"
	case IPNSRecordItemType:
		return "IPNS_RECORD"
	case DNSLinkItemType:
		return "IPNS_RECORD"
	case Libp2pPeerIDItemType:
		return "LIBP2P_PEER_ID"
	case AvailDASCellItemType:
		return "AVAIL_DAS_CELL"
	case CelestiaDHTNamesSpaceItemType:
		return "CELESTIA_NAMESPACE"
	default:
		return "UNKNOWN"
	}
}

const (
	UnknownItemType SamplingItemType = iota
	// IPFS + IPNS + Libp2p
	IPFSCidItemType
	IPNSRecordItemType
	DNSLinkItemType
	Libp2pPeerIDItemType
	// Avail
	AvailDASCellItemType
	// Celestia
	CelestiaDHTNamesSpaceItemType
)

func SamplingItemFromStrin(s string) SamplingItemType {
	switch s {
	case "UNKNOWN":
		return UnknownItemType
	case "IPFS_CID":
		return IPFSCidItemType
	case "IPNS_RECORD":
		return IPNSRecordItemType
	case "DNS_LINK":
		return DNSLinkItemType
	case "LIBP2P_PEER_ID":
		return Libp2pPeerIDItemType
	case "AVAIL_DAS_CELL":
		return AvailDASCellItemType
	case "CELESTIA_NAMESPACE":
		return CelestiaDHTNamesSpaceItemType
	default:
		return UnknownItemType
	}
}

var ErrorNotValidKeyType error = fmt.Errorf("not a valid sampling key type")

func ParseDHTKeyType(t SamplingItemType, key string) (any, error) {
	switch t {
	case UnknownItemType:
		return nil, ErrorNotValidKeyType
	case IPFSCidItemType:
		return cid.Decode(key)
	case IPNSRecordItemType:
		return ComposeIpnsKey(key)
	case DNSLinkItemType:
		return key, nil
	case Libp2pPeerIDItemType:
		return peer.Decode(key)
	case AvailDASCellItemType:
		return AvailKeyFromString(key)
	case CelestiaDHTNamesSpaceItemType:
		return key, nil
	default:
		return nil, ErrorNotValidKeyType
	}
}
