package zerocopy

import (
	crand "crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAtob(t *testing.T) {
	for i := 0; i < 1000; i++ {
		s := randstr(i)
		require.Equal(t, []byte(s), UnsafeAtob(s))
	}
}

func TestBtoa(t *testing.T) {
	for i := 0; i < 1000; i++ {
		s := randstr(i)
		require.Equal(t, s, UnsafeBtoa([]byte(s)))
	}
}

func randstr(n int) string {
	b := make([]byte, n)
	crand.Read(b)
	return string(b)
}
