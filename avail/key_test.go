package avail

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AvailKey(t *testing.T) {
	block := uint64(123)
	row := uint64(12)
	column := uint64(54)

	availKey, err := KeyFromString(fmt.Sprintf("%d:%d:%d", block, row, column))
	require.NoError(t, err)

	require.Equal(t, availKey.Block, block)
	require.Equal(t, availKey.Row, row)
	require.Equal(t, availKey.Column, column)
}
