package ratelimit

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"
)

type tokenReq struct {
	l       sync.Mutex
	size    int
	cancelc <-chan struct{}
	donec   chan struct{}
}

func (r *tokenReq) iscanceled() bool {
	select {
	case <-r.cancelc:
		return true
	default:
		return false
	}
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

func (h *tokenReqHeap) Len() int           { return len(*h) }
func (h *tokenReqHeap) Less(i, j int) bool { return (*h)[i].size < (*h)[j].size }
func (h *tokenReqHeap) Swap(i, j int)      { (*h)[i], (*h)[j] = (*h)[j], (*h)[i] }

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

type tokenReqGenerations struct {
	generations []*tokenReqHeap
	start       int
	end         int
	size        int
	cap         int
}

func newTokenReqGenerations() *tokenReqGenerations {
	// Eventually, the length of the ring will stay constant.
	g := &tokenReqGenerations{
		generations: []*tokenReqHeap{},
		start:       -1,
		end:         -1,
		size:        0,
		cap:         0,
	}
	g.newgeneration()
	return g
}

func (g *tokenReqGenerations) String() string {
	generations := []int{}
	for _, h := range g.generations {
		generations = append(generations, h.Len())
	}
	return fmt.Sprintf("{start:%d, end:%d, size:%d, cap:%d, g:%v}", g.start, g.end, g.size, g.cap, generations)
}

func (g *tokenReqGenerations) newgeneration() {
	if g.size > 0 && g.generations[g.end].Len() == 0 {
		return
	}

	if g.size < g.cap {
		g.end = (g.end + 1) % g.cap
		g.size++
	} else {
		if g.end < g.start {
			oldg := g.generations
			g.generations = make([]*tokenReqHeap, g.cap+1)
			copy(g.generations, oldg[g.start:])
			copy(g.generations[g.cap-g.start:], oldg[:g.end+1])
			g.generations[g.cap] = &tokenReqHeap{}
		} else {
			g.generations = append(g.generations, &tokenReqHeap{})
		}
		g.end = g.cap
		g.cap++
		g.size++
		g.start = 0
	}
}

func (g *tokenReqGenerations) newtokenreq(req *tokenReq) {
	heap.Push(g.generations[g.end], req)
}

func (g *tokenReqGenerations) visitoldest(visitor func(req *tokenReq) bool) {
	if g.size == 0 {
		return
	}

	oldest := g.generations[g.start]
	for oldest.Len() > 0 {
		if !visitor((*oldest)[0]) {
			break
		}
		heap.Pop(oldest)

		if oldest.Len() == 0 && g.start != g.end {
			g.start = (g.start + 1) % g.cap
			g.size--
		}
		oldest = g.generations[g.start]
	}
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
	// XXX(damnever): maybe FIFO is good enough..
	// TODO(damnever): more tests..
	pending := newTokenReqGenerations()
	visitor := func(req *tokenReq) bool {
		if req.iscanceled() {
			return true
		}
		if bucket >= req.size {
			bucket -= req.markdone()
			return true
		}
		return false
	}

	for {
		select {
		case <-l.stopc:
			pending.visitoldest(visitor)
			return
		case <-ticker.C:
			if x := bucket + token; x < l.limit {
				bucket = x
			} else {
				bucket = l.limit
			}
			pending.newgeneration()
			pending.visitoldest(visitor)
		case req := <-l.reqc:
			if req.size > bucket {
				pending.newtokenreq(req)
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
