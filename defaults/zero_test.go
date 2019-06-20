package defaults

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTIfZero(t *testing.T) {
	require.Equal(t, 1, IntIfZero(0, 1))
	require.Equal(t, int8(1), Int8IfZero(0, 1))
	require.Equal(t, int16(1), Int16IfZero(0, 1))
	require.Equal(t, int32(1), Int32IfZero(0, 1))
	require.Equal(t, int64(1), Int64IfZero(0, 1))
	require.Equal(t, uint(1), UintIfZero(0, 1))
	require.Equal(t, uint8(1), Uint8IfZero(0, 1))
	require.Equal(t, uint16(1), Uint16IfZero(0, 1))
	require.Equal(t, uint32(1), Uint32IfZero(0, 1))
	require.Equal(t, uint64(1), Uint64IfZero(0, 1))
	require.Equal(t, float32(1), Float32IfZero(0, 1))
	require.Equal(t, float64(1), Float64IfZero(0, 1))
	require.Equal(t, time.Second, DurationIfZero(0, time.Second))
	require.Equal(t, "good", StringIfEmpty("", "good"))
	now := time.Now()
	require.Equal(t, now, TimeIfZero(time.Time{}, now))
}
