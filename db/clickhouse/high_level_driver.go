package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/probe-lab/akai/config"
)

func (s *ClickHouseDB) getHighLevelConnection(
	_ context.Context,
	conDetails *config.DatabaseDetails,
) (driver.Conn, error) {

	var opts clickhouse.Options
	switch s.instanceType {
	case ClickhouseLocalInstance, ClickhouseReplicatedInstance:
		opts = clickhouse.Options{
			Addr: []string{conDetails.Address},
			Auth: clickhouse.Auth{
				Database: conDetails.Database,
				Username: conDetails.User,
				Password: conDetails.Password,
			},
			Debug: false,
			Debugf: func(format string, v ...any) {
				fmt.Printf(format, v)
			},
			Settings: clickhouse.Settings{
				"max_execution_time": 60,
			},
			Compression: &clickhouse.Compression{
				Method: clickhouse.CompressionLZ4,
			},
			DialTimeout:          time.Second * 30,
			MaxOpenConns:         5,
			MaxIdleConns:         5,
			ConnMaxLifetime:      time.Duration(10) * time.Minute,
			ConnOpenStrategy:     clickhouse.ConnOpenInOrder,
			BlockBufferSize:      10,
			MaxCompressionBuffer: 10240,
			ClientInfo: clickhouse.ClientInfo{ // optional, please see Client info section in the README.md
				Products: []struct {
					Name    string
					Version string
				}{
					{Name: config.ClientName, Version: config.ClientVersion},
				},
			},
		}
	default:
		return nil, fmt.Errorf("%s clickhouse doesn't support low-level connections", s.instanceType)
	}

	if conDetails.TLSrequired {
		opts.TLS = &tls.Config{}
	}
	// get a connection to the high level driver
	conn, err := clickhouse.Open(&opts)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
