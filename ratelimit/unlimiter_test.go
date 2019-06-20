package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUnlimiter(t *testing.T) {
	l := NewUnlimiter()

	wg := sync.WaitGroup{}
	start := time.Now()

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.Nil(t, l.Take(context.TODO(), 11111111))
		}()
	}

	wg.Wait()
	l.Close()

	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("unbelievable")
	}
}
