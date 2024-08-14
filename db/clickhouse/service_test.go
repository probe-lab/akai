package clickhouse

import (
	"context"
	"testing"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	"github.com/stretchr/testify/require"
)

func Test_ClickhouseInitialization(t *testing.T) {
	// variables
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = generateClickhouseDatabase(t, mainCtx, make(map[string]struct{}, 0))
}

func generateClickhouseDatabase(t *testing.T, ctx context.Context, tables map[string]struct{}) *ClickHouseDB {
	conDetails := ConnectionDetails{
		Driver:   "clickhouse",
		Address:  "127.0.0.1:9000",
		User:     "username",
		Password: "password",
		Database: "akai_test",
		Params:   "",
	}
	network := db.Network{
		NetworkID:   0,
		Protocol:    config.ProtocolLocalCustom,
		NetworkName: config.NetworkNameLocalCustom,
	}
	// generate new connection
	clickhouseDB, err := NewClickHouseDB(conDetails, network)
	require.NoError(t, err)

	err = clickhouseDB.Init(ctx, tables)
	require.NoError(t, err)

	return clickhouseDB
}
