package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestUnlimiter(t *testing.T) {
	l := NewUnlimiter()

	wg := sync.WaitGroup{}
	start := time.Now()

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := l.Take(context.TODO(), 11111111); err != nil {
				t.Fatalf("expect nil, got: %v", err)
			}
		}()
	}

	wg.Wait()
	l.Close()

	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("unbelievable")
	}
}
