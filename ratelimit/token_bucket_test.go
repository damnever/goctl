package ratelimit

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenReqGenerations(t *testing.T) {
	g := newTokenReqGenerations()
	g.newgeneration()
	assert(t, g.cap, 1)
	assert(t, g.size, 1)
	assert(t, g.start, 0)
	assert(t, g.end, 0)

	g.newtokenreq(&tokenReq{})
	g.newtokenreq(&tokenReq{})
	g.newtokenreq(&tokenReq{})
	assert(t, g.cap, 1)
	assert(t, g.size, 1)
	assert(t, g.start, 0)
	assert(t, g.end, 0)

	g.newgeneration()
	assert(t, g.cap, 2)
	assert(t, g.size, 2)
	assert(t, g.start, 0)
	assert(t, g.end, 1)
	g.newtokenreq(&tokenReq{})
	g.newtokenreq(&tokenReq{})
	g.newtokenreq(&tokenReq{})

	i := 0
	g.visitoldest(func(req *tokenReq) bool {
		if i++; i > 3 {
			return false
		}
		return true
	})
	assert(t, g.cap, 2)
	assert(t, g.size, 1)
	assert(t, g.start, 1)
	assert(t, g.end, 1)

	g.newgeneration()
	assert(t, g.cap, 2)
	assert(t, g.size, 2)
	assert(t, g.start, 1)
	assert(t, g.end, 0)
	g.newtokenreq(&tokenReq{})
	g.newtokenreq(&tokenReq{})
	g.newtokenreq(&tokenReq{})

	i = 0
	g.visitoldest(func(req *tokenReq) bool {
		if i++; i > 3 {
			return false
		}
		return true
	})
	assert(t, g.cap, 2)
	assert(t, g.size, 1)
	assert(t, g.start, 0)
	assert(t, g.end, 0)

	g.visitoldest(func(req *tokenReq) bool {
		return true
	})
	assert(t, g.cap, 2)
	assert(t, g.size, 1)
	assert(t, g.start, 0)
	assert(t, g.end, 0)

	g.newtokenreq(&tokenReq{})
	g.newgeneration()
	g.newtokenreq(&tokenReq{})
	i = 0
	g.visitoldest(func(req *tokenReq) bool {
		if i++; i > 1 {
			return false
		}
		return true
	})
	g.newgeneration()
	g.newtokenreq(&tokenReq{})
	g.newtokenreq(&tokenReq{})
	g.newgeneration()
	g.newtokenreq(&tokenReq{})
	g.newtokenreq(&tokenReq{})
	g.newtokenreq(&tokenReq{})
	assert(t, g.cap, 3)
	assert(t, g.size, 3)
	assert(t, g.end, 2)
	assert(t, g.start, 0)
	for i, g := range g.generations {
		assert(t, g.Len(), i+1)
	}
}

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
	// 50ms for CI..
	if !(elapsed <= time.Second+50*time.Millisecond && elapsed >= time.Second) {
		t.Fatalf("expect time range[1s, 1s+50ms], got: %v", elapsed)
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
	// 50ms for CI..
	if !(elapsed <= time.Second+50*time.Millisecond && elapsed >= time.Second) {
		t.Fatalf("expect time range[1s, 1s+50ms], got: %v", elapsed)
	}
}

func TestNoStarving(t *testing.T) {
	MB200 := 200 * (1 << 20)
	l := NewTokenBucketRateLimiter(MB200)
	defer l.Close()

	ctx := context.TODO()
	minsize := MB200/int(time.Second/time.Millisecond) - 1
	sizes := []int{minsize, minsize*2 + 3}
	ngo, sizego := 32, MB200/16

	var small, big int32
	wg := sync.WaitGroup{}
	for i := 0; i < ngo; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for count, next := 0, 0; count < sizego; next = (next + 1) % 2 {
				func() {
					cctx, cancel := context.WithTimeout(ctx, 60*time.Millisecond)
					defer cancel()
					size := r.Intn(sizes[next])
					if size == 0 {
						size = sizes[next]
					}
					if l.Take(cctx, size) == nil {
						count += size
						if next == 0 {
							atomic.AddInt32(&small, 1)
						} else {
							atomic.AddInt32(&big, 1)
						}
					} else {
						t.Fatalf("unbelivable")
					}
				}()
			}
		}()
	}
	wg.Wait()

	if big >= small*2 || big*2 <= small {
		t.Fatalf("somebody is starving: %d vs %d", small, big)
	}
	fmt.Printf("B(%d) vs S(%v)\n", big, small)
}

func TestConcurrentOPS(t *testing.T) {
	MB200 := 200 * (1 << 20)
	l := NewTokenBucketRateLimiter(MB200)

	ctx := context.TODO()
	wg := sync.WaitGroup{}
	start := time.Now()

	ngo, sizego := 64, MB200/8
	var canceled int32
	for i := 0; i < ngo; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for count := 0; count < sizego; {
				func() {
					timeout := time.Duration(r.Intn(66)) * time.Millisecond
					if timeout == 0 {
						timeout = 1 * time.Second
					}
					cctx, cancel := context.WithTimeout(ctx, timeout)
					defer cancel()

					size := r.Intn(1 << 15)
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

	minexpect := time.Duration(ngo*sizego/MB200) * time.Second
	// 4500ms for the CI.. ~8.03s in my computer
	if !(elapsed >= minexpect && elapsed <= minexpect+4500*time.Millisecond) {
		t.Fatalf("expect time range[%v, %v+4500ms], got: %v", minexpect, minexpect, elapsed)
	}
	if canceled < 10 {
		t.Fatalf("expect 10 canceled, got: %d", canceled)
	}
}

func assert(t *testing.T, actual interface{}, expect interface{}) {
	_, fileName, line, _ := runtime.Caller(1)
	if actual != expect {
		t.Fatalf("expect %v, got %v at (%v:%v)\n", expect, actual, fileName, line)
	}
}
