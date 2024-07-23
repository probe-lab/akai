package amino

import "github.com/ipfs/go-cid"

type Cid cid.Cid

func (c *Cid) Cid() cid.Cid {
	return cid.Cid(*c)
}

func CidFromString(s string) (Cid, error) {
	c, err := cid.Decode(s)
	if err != nil {
		return Cid{}, err
	}
	return Cid(c), nil
}
