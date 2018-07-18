package ratelimit

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

type tokenReq struct {
	l       sync.Mutex
	size    int
	cancelc <-chan struct{}
	donec   chan struct{}
}

func (r *tokenReq) markdone() (size int) {
	r.l.Lock()
	defer r.l.Unlock()
	select {
	case <-r.cancelc:
	default:
		if r.donec != nil {
			close(r.donec)
			size = r.size
		}
	}
	return
}

func (r *tokenReq) isdone() bool {
	r.l.Lock()
	defer r.l.Unlock()
	select {
	case <-r.donec:
		return true
	default:
		r.donec = nil
		return false
	}
}

type tokenReqHeap []*tokenReq

func (h tokenReqHeap) Len() int           { return len(h) }
func (h tokenReqHeap) Less(i, j int) bool { return h[i].size < h[j].size }
func (h tokenReqHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *tokenReqHeap) Push(x interface{}) {
	*h = append(*h, x.(*tokenReq))
}

func (h *tokenReqHeap) Pop() interface{} {
	last := len(*h) - 1
	x := (*h)[last]
	(*h)[last] = nil
	*h = (*h)[0:last]
	return x
}

type tokenBucketRateLimiter struct {
	limit int
	reqc  chan *tokenReq
	stopc chan struct{}
	donec chan struct{}
}

// NewTokenBucketRateLimiter creates a new token bucket RateLimiter.
//
// QPS:
//     l := NewTokenBucketRateLimiter(1000) // 1000 queries per second
//     defer l.Close()
//     err := l.Take(ctx, 1) // take a token
//
// BPS:
//     l := NewTokenBucketRateLimiter(200*(1<<20)) // 200MB per second
//     defer l.Close()
//     err := l.Take(ctx, 1<<20) // take 1MB
func NewTokenBucketRateLimiter(limit int) RateLimiter {
	l := &tokenBucketRateLimiter{
		limit: limit,
		reqc:  make(chan *tokenReq, 32),
		stopc: make(chan struct{}),
		donec: make(chan struct{}),
	}
	go l.scheduling()
	return l
}

func (l *tokenBucketRateLimiter) scheduling() {
	defer close(l.donec)
	interval := time.Second / time.Duration(l.limit)
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	bucket := 0
	token := l.limit / int(time.Second/interval) // Approximately..
	pending := &tokenReqHeap{}                   // XXX(damnever): FIFO or memory-like allocation algorithm????
	clearPending := func() {
		for pending.Len() > 0 && bucket >= (*pending)[0].size {
			req := heap.Pop(pending).(*tokenReq)
			bucket -= req.markdone()
		}
	}

	for {
		select {
		case <-l.stopc:
			clearPending()
			return
		case <-ticker.C:
			if x := bucket + token; x < l.limit {
				bucket = x
			} else {
				bucket = l.limit
			}
			clearPending()
		case req := <-l.reqc:
			if req.size > bucket {
				heap.Push(pending, req)
			} else {
				bucket -= req.markdone()
			}
		}
	}
}

func (l *tokenBucketRateLimiter) Take(ctx context.Context, size int) error {
	cancelc := ctx.Done()
	req := &tokenReq{
		size:    size,
		cancelc: cancelc,
		donec:   make(chan struct{}),
	}

	select {
	case <-cancelc:
		return ctx.Err()
	case l.reqc <- req:
	}

	select {
	case <-cancelc:
		if !req.isdone() {
			return ctx.Err()
		}
	case <-req.donec:
	}
	return nil
}

func (l *tokenBucketRateLimiter) Close() error {
	close(l.stopc)
	<-l.donec
	return nil
}
