package circuitbreaker

import (
	"time"
)

type rollingCounter struct {
	values   []int
	size     int
	lastidx  int
	lasttime time.Time
	interval time.Duration
}

func newRollingCounter(window time.Duration, now time.Time) *rollingCounter {
	interval := time.Second
	size := int(window / interval)
	if size < 10 {
		size = 10
		interval = window / time.Duration(size)
	}
	return &rollingCounter{
		values:   make([]int, size, size),
		size:     size,
		lastidx:  0,
		lasttime: now,
		interval: interval,
	}
}

func (rc *rollingCounter) Count(now time.Time) int {
	rc.advance(now)

	count := 0
	for _, v := range rc.values {
		count += v
	}
	return count
}

func (rc *rollingCounter) Incr(now time.Time) {
	rc.advance(now)
	rc.values[rc.lastidx]++ // that's..
}

func (rc *rollingCounter) advance(now time.Time) {
	elapsed := int(now.Sub(rc.lasttime) / rc.interval)
	if elapsed < 1 {
		return
	}

	rc.lasttime = now
	if elapsed > rc.size {
		elapsed = rc.size
	}
	oldidx := rc.lastidx + 1
	lastidx := rc.lastidx + elapsed
	rc.lastidx = lastidx % rc.size
	for ; oldidx <= lastidx; oldidx++ {
		rc.values[oldidx%rc.size] = 0
	}
}
