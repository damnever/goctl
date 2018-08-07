package circuitbreaker

import (
	"runtime"
	"testing"
	"time"
)

func TestRollingCounter(t *testing.T) {
	now := time.Now()
	rc := newRollingCounter(time.Second, now)
	n := rc.size
	duration := rc.interval

	count := 0
	for i := 1; i <= n; i++ {
		for j := 0; j < i; j++ {
			rc.Incr(now)
		}
		count += i
		assert(t, rc.Count(now), count)
		if i != n {
			now = now.Add(duration)
		}
	}

	count = rc.Count(now)
	now = now.Add(duration)
	for i := 1; i <= n; i++ {
		rc.Incr(now)
		rc.Incr(now)
		count = count - i + 2
		assert(t, rc.Count(now), count)
		if i != n {
			now = now.Add(duration)
		}
	}

	now = now.Add(3*duration + duration/2)
	assert(t, rc.Count(now), (n-3)*2)
	rc.Incr(now)
	assert(t, rc.Count(now), (n-3)*2+1)

	now = now.Add(time.Duration(n-2) * duration)
	assert(t, rc.Count(now), 1)

	now = now.Add(2 * duration)
	assert(t, rc.Count(now), 0)
}

func assert(t *testing.T, actual interface{}, expect interface{}) {
	if actual != expect {
		_, fileName, line, _ := runtime.Caller(1)
		t.Fatalf("expect %v, got %v at (%v:%v)\n", expect, actual, fileName, line)
	}
}
