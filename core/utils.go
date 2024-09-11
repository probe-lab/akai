package core

import "github.com/libp2p/go-libp2p/core/peer"

func ListPeerIDsFromAddrsInfos(addrs []peer.AddrInfo) []peer.ID {
	peerIDs := make([]peer.ID, len(addrs))

	for i, addr := range addrs {
		peerIDs[i] = addr.ID
	}
	return peerIDs
}
