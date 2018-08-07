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

func TestTokenReqRing(t *testing.T) {
	r := newTokenReqRing()
	r.add(&tokenReq{})
	r.add(&tokenReq{})
	assert(t, r.cap, 2)
	assert(t, r.length(), 2)
	assert(t, r.start, 0)
	assert(t, r.end, 1)
	r.add(&tokenReq{})
	assert(t, r.cap, 4)
	assert(t, r.length(), 3)
	assert(t, r.start, 0)
	assert(t, r.end, 2)
	r.add(&tokenReq{})
	r.popfirst()
	assert(t, r.size, 3)
	assert(t, r.start, 1)
	r.popfirst()
	r.popfirst()
	r.popfirst()
	assert(t, r.length(), 0)
	assert(t, r.start, -1)
	assert(t, r.start, -1)
	r.reqs[0] = &tokenReq{size: 3}
	r.reqs[1] = &tokenReq{size: 4}
	r.reqs[2] = &tokenReq{size: 1}
	r.reqs[3] = &tokenReq{size: 2}
	r.end = 1
	r.start = 2
	r.size = 4
	r.add(&tokenReq{size: 5})
	assert(t, r.end, 4)
	for i := 0; i < r.end; i++ {
		assert(t, r.reqs[i].size, i+1)
	}
	r.popfirst()
	assert(t, r.fisrt().size, 2)
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
	if !(elapsed <= time.Second+33*time.Millisecond && elapsed >= time.Second-10*time.Millisecond) {
		t.Fatalf("expect time range[1s, 1s+33ms], got: %v", elapsed)
	}
}

func TestBPSLikeRateLimit(t *testing.T) {
	MB10 := 10 * (1 << 20)
	l := NewTokenBucketRateLimiter(MB10)
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
	if !(elapsed <= time.Second+33*time.Millisecond && elapsed >= time.Second-10*time.Millisecond) {
		t.Fatalf("expect time range[1s, 1s+33ms], got: %v", elapsed)
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
					cctx, cancel := context.WithTimeout(ctx, 33*time.Millisecond)
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
						// panic("unbelievable")
					}
				}()
			}
		}()
	}
	wg.Wait()

	if (big >= small*2 || big*2 <= small) || small-big > int32(float64((big+big)/2)*0.08) {
		t.Fatalf("somebody is starving: %d vs %d", small, big)
	}
	fmt.Printf("B(%d) vs S(%v)\n", big, small)
}

func TestConcurrentOPS(t *testing.T) {
	MB512 := 512 * (1 << 20)
	l := NewTokenBucketRateLimiter(MB512)

	ctx := context.TODO()
	wg := sync.WaitGroup{}
	start := time.Now()

	ngo, sizego := 256, MB512/64
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

					size := r.Intn(5234790) + 8092
					if size == 0 {
						size = 1
					}
					if l.Take(cctx, size) == nil {
						count += size
					} else {
						atomic.AddInt32(&canceled, 1)
						time.Sleep(10 * time.Millisecond)
					}
				}()
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)
	l.Close()

	minexpect := time.Duration(ngo*sizego/MB512) * time.Second
	if !(elapsed >= minexpect && elapsed <= minexpect+2000*time.Millisecond) {
		t.Fatalf("expect time range[%v, %v+2000ms], got: %v", minexpect, minexpect, elapsed)
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
