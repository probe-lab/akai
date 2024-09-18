package clickhouse

import (
	"context"
	"testing"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
	"github.com/stretchr/testify/require"
)

func Test_ClickhouseInitialization(t *testing.T) {
	// variables
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = generateClickhouseDatabase(t, mainCtx, make(map[string]struct{}, 0))
}

func generateClickhouseDatabase(t *testing.T, ctx context.Context, tables map[string]struct{}) *ClickHouseDB {
	conDetails := DefaultClickhouseConnectionDetails
	network := models.Network{
		NetworkID:   0,
		Protocol:    config.ProtocolLocal,
		NetworkName: config.NetworkNameCustom,
	}
	// generate new connection
	clickhouseDB, err := NewClickHouseDB(conDetails, network)
	require.NoError(t, err)

	err = clickhouseDB.Init(ctx, tables)
	require.NoError(t, err)

	return clickhouseDB
}
