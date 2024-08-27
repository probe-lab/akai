package config

import "time"

type DHTHostType int8

const (
	DHTClient DHTHostType = iota
	DHTServer
)

type CommonDHTOpts struct {
	ID          int
	IP          string
	Port        int64
	DialTimeout time.Duration
	DHTMode     DHTHostType
	UserAgent   string
}
