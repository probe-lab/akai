package models

type Root []byte

func (r Root) String() string {
	return string(r)
}

func (r Root) Len() int {
	return len(r)
}
