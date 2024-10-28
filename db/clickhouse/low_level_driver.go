package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/probe-lab/akai/config"
)

type lowLevelConn struct {
	tag string
	sync.Mutex
	conn *ch.Client
}

func (c *lowLevelConn) persist(
	ctx context.Context,
	baseQuery string,
	tableName string,
	input proto.Input,
) error {
	c.Lock()
	err := c.conn.Do(ctx, ch.Query{
		Body:  fmt.Sprintf(baseQuery, tableName),
		Input: input,
	})
	c.Unlock()
	return err
}

func (c *lowLevelConn) close() error {
	c.Lock()
	err := c.conn.Close()
	c.Unlock()
	return err
}

func (s *ClickHouseDB) getLowLevelConnection(
	ctx context.Context,
	conDetails *config.DatabaseDetails,
	tag string,
) (*lowLevelConn, error) {
	opts := ch.Options{
		Address:  conDetails.Address,
		Database: conDetails.Database,
		User:     conDetails.User,
		Password: conDetails.Password,
	}

	if conDetails.TLSrequired {
		opts.TLS = &tls.Config{}
	}

	lowConn, err := ch.Dial(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &lowLevelConn{
		tag:  tag,
		conn: lowConn,
	}, nil
}
