package circuitbreaker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
		require.Equal(t, count, rc.Count(now))
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
		require.Equal(t, count, rc.Count(now))
		if i != n {
			now = now.Add(duration)
		}
	}

	now = now.Add(3*duration + duration/2)
	require.Equal(t, (n-3)*2, rc.Count(now))
	rc.Incr(now)
	require.Equal(t, (n-3)*2+1, rc.Count(now))

	now = now.Add(time.Duration(n-2) * duration)
	require.Equal(t, 1, rc.Count(now))

	now = now.Add(2 * duration)
	require.Equal(t, 0, rc.Count(now))
}
