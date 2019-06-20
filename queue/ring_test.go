package queue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRing(t *testing.T) {
	r := NewRing()
	r.Append(1)
	r.Append(2)
	require.Equal(t, 2, r.cap)
	require.Equal(t, 2, r.Len())
	require.Equal(t, 0, r.head)
	require.Equal(t, 1, r.tail)

	r.Append(3)
	require.Equal(t, 4, r.cap)
	require.Equal(t, r.Len(), 3)
	require.Equal(t, 0, r.head)
	require.Equal(t, 2, r.tail)
	r.Append(4)
	require.Equal(t, 1, r.Pop())
	require.Equal(t, 3, r.size)
	require.Equal(t, 1, r.head)
	len := r.Len()
	for i := 2; i <= 4; i++ {
		require.Equal(t, i, r.Pop())
		len--
		require.Equal(t, len, r.Len())
	}

	require.Equal(t, 0, r.Len())
	require.Equal(t, nil, r.Pop())
	require.Equal(t, nil, r.Peek())

	require.Equal(t, 0, r.head)
	require.Equal(t, -1, r.tail)
	r.items[0] = 3
	r.items[1] = 4
	r.items[2] = 1
	r.items[3] = 2
	r.tail = 1
	r.head = 2
	r.size = 4

	require.Equal(t, 1, r.Peek())
	r.Append(5)
	require.Equal(t, 4, r.tail)
	for i := 0; i < r.tail; i++ {
		require.Equal(t, i+1, r.Pop())
	}
}
