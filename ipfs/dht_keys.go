package ipfs

import "github.com/ipfs/go-cid"

func CidFromString(s string) (cid.Cid, error) {
	return cid.Decode(s)
}
