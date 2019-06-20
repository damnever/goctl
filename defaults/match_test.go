package defaults

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTIfNotMatch(t *testing.T) {
	require.Equal(t, 1, IntIfNotMatch(0, 1, "N>=1"))
	require.Equal(t, int8(1), Int8IfNotMatch(0, 1, "N!=0"))
	require.Equal(t, int16(1), Int16IfNotMatch(0, 1, "N%2!=0"))
	require.Equal(t, int32(1), Int32IfNotMatch(0, 1, "N/2>0"))
	require.Equal(t, int64(-1), Int64IfNotMatch(0, -1, "N*5<0"))
	require.Equal(t, uint(1), UintIfNotMatch(0, 1, "N+1>2"))
	require.Equal(t, uint8(1), Uint8IfNotMatch(2, 1, "N-1==0"))
	require.Equal(t, uint16(1), Uint16IfNotMatch(0, 1, "!(N==0)"))
	require.Equal(t, uint32(1), Uint32IfNotMatch(0, 1, "N*2!=0"))
	require.Equal(t, uint64(1), Uint64IfNotMatch(0, 1, "N==1"))
	require.Equal(t, float32(0.9), Float32IfNotMatch(0, 0.9, "N>=0.1&&N<=0.9"))
	require.Equal(t, float64(0.8), Float64IfNotMatch(0, 0.8, "!(N<0.1)||N>0.9"))
	require.Equal(t, time.Second, DurationIfNotMatch(0, time.Second, "N>0"))
	require.Equal(t, "good", StringIfNotMatch("bad", "good", "^g"))
}

func TestTIfNotMatchFunc(t *testing.T) {
	require.Equal(t, 1, IntIfNotMatchFunc(0, 1, func(i int) bool { return i >= 1 }))
	require.Equal(t, int8(1), Int8IfNotMatchFunc(0, 1, func(i int8) bool { return i != 0 }))
	require.Equal(t, int16(1), Int16IfNotMatchFunc(0, 1, func(i int16) bool { return i%2 != 0 }))
	require.Equal(t, int32(1), Int32IfNotMatchFunc(0, 1, func(i int32) bool { return i/2 > 0 }))
	require.Equal(t, int64(-1), Int64IfNotMatchFunc(0, -1, func(i int64) bool { return i*5 < 0 }))
	require.Equal(t, uint(1), UintIfNotMatchFunc(0, 1, func(i uint) bool { return i+1 > 2 }))
	require.Equal(t, uint8(1), Uint8IfNotMatchFunc(2, 1, func(i uint8) bool { return i-1 == 0 }))
	require.Equal(t, uint16(1), Uint16IfNotMatchFunc(0, 1, func(i uint16) bool { return !(i == 0) }))
	require.Equal(t, uint32(1), Uint32IfNotMatchFunc(0, 1, func(i uint32) bool { return i*2 != 0 }))
	require.Equal(t, uint64(1), Uint64IfNotMatchFunc(0, 1, func(i uint64) bool { return i == 1 }))
	require.Equal(t, float32(0.9), Float32IfNotMatchFunc(0, 0.9, func(i float32) bool { return i >= 0.1 && i <= 0.9 }))
	require.Equal(t, float64(0.8), Float64IfNotMatchFunc(0, 0.8, func(i float64) bool { return !(i < 0.1) || i > 0.9 }))
	require.Equal(t, time.Second, DurationIfNotMatchFunc(0, time.Second, func(i time.Duration) bool { return i > 0 }))
	require.Equal(t, "good", StringIfNotMatchFunc("bad", "good", func(i string) bool { return strings.HasPrefix(i, "g") }))
	now := time.Now()
	require.Equal(t, now, TimeIfNotMatchFunc(time.Time{}, now, func(i time.Time) bool { return !i.IsZero() }))
}
