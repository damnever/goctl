package ratelimit

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestQPSLikeRateLimit(t *testing.T) {
	l := NewTokenBucketRateLimiter(1000)
	defer l.Close()

	start := time.Now()
	for i := 0; i < 1000; i++ {
		if err := l.Take(context.TODO(), 1); err != nil {
			t.Fatalf("expect nil, got: %v", err)
		}
	}

	elapsed := time.Since(start)
	if !(elapsed <= time.Second+23*time.Millisecond && elapsed >= time.Second) {
		t.Fatalf("expect time range[1s, 1s+23ms], got: %v", elapsed)
	}
}

func TestBPSLikeRateLimit(t *testing.T) {
	MB10 := 10 * (1 << 20)
	l := NewTokenBucketRateLimiter(MB10) // 10MB
	defer l.Close()

	start := time.Now()
	count := 0
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for count < MB10 {
		size := r.Intn((1 << 10))
		if size == 0 {
			size = 1
		}
		if err := l.Take(context.TODO(), size); err != nil {
			t.Fatalf("expect nil, got: %v", err)
		}
		count += size
	}

	elapsed := time.Since(start)
	if !(elapsed <= time.Second+23*time.Millisecond && elapsed >= time.Second) {
		t.Fatalf("expect time range[1s, 1s+23ms], got: %v", elapsed)
	}
}

func TestConcurrentOPS(t *testing.T) {
	MB200 := 200 * (1 << 20)
	l := NewTokenBucketRateLimiter(MB200)

	ctx := context.TODO()
	wg := sync.WaitGroup{}
	start := time.Now()

	n, persize := 64, MB200/8
	var canceled int32
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for count := 0; count < persize; {
				func() {
					cctx, cancel := context.WithTimeout(ctx, time.Duration(r.Intn(66))*time.Millisecond)
					defer cancel()
					size := r.Intn(1 << 14)
					if size == 0 {
						size = 1
					}
					if l.Take(cctx, size) == nil {
						count += size
					} else {
						atomic.AddInt32(&canceled, 1)
					}
				}()
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)
	l.Close()

	minexpect := time.Duration(n*persize/MB200) * time.Second
	// 3000ms for the CI.. ~8.03s in my computer
	if !(elapsed >= minexpect && elapsed <= minexpect+3000*time.Millisecond) {
		t.Fatalf("expect time range[%v, %v+1500ms], got: %v", minexpect, minexpect, elapsed)
	}
	if canceled < 10 {
		t.Fatalf("expect 10 canceled, got: %d", canceled)
	}
}
