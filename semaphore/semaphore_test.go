package semaphore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSemaphore(t *testing.T) {
	ctx := context.TODO()
	sem := NewSemaphore(2)
	require.Nil(t, sem.Acquire(ctx))
	require.Nil(t, sem.Acquire(ctx))

	acquired := make(chan struct{})
	go func() {
		require.Nil(t, sem.Acquire(ctx))
		close(acquired)
		require.Nil(t, sem.Release())
	}()
	select {
	case <-acquired:
		t.Fatalf("limit exceed")
	case <-time.After(10 * time.Millisecond):
	}
	require.Nil(t, sem.Release())
	<-acquired
	require.Nil(t, sem.Release())
	require.Equal(t, ErrOpMismatch, sem.Release())
}
