package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetry(t *testing.T) {
	backoff := 20 * time.Millisecond
	{
		cnt := 0
		now := time.Now()
		err := Retry(ConstantBackoffs(5, backoff), func() (State, error) {
			cnt++
			return Continue, ErrNeedRetry
		})
		require.NotNil(t, err)
		require.Equal(t, 6, cnt)
		elapsed := time.Now().Sub(now)
		require.True(t, elapsed > backoff*5)
		require.True(t, elapsed < backoff*6)
	}
	{
		cnt := 0
		now := time.Now()
		err := Retry(ExponentialBackoffs(5, backoff), func() (State, error) {
			cnt++
			return Continue, nil
		})
		require.Nil(t, err)
		require.Equal(t, 1, cnt)
		require.True(t, time.Now().Sub(now) < backoff)
	}
	{
		cnt := 0
		err := Retry(ZeroBackoffs(5), func() (State, error) {
			cnt++
			if cnt == 2 {
				return StopWithErr, ErrNeedRetry
			}
			return Continue, ErrNeedRetry
		})
		require.Equal(t, 2, cnt)
		require.Equal(t, ErrNeedRetry, err)
	}
	{
		cnt := 0
		err := Retry(ZeroBackoffs(5), func() (State, error) {
			cnt++
			if cnt == 2 {
				return StopWithNil, ErrNeedRetry
			}
			return Continue, ErrNeedRetry
		})
		require.Equal(t, 2, cnt)
		require.Nil(t, err)
	}
}

func TestBackoffFactory(t *testing.T) {
	{
		backoffs := ZeroBackoffs(3)
		require.Equal(t, 3, len(backoffs))
		for _, v := range backoffs {
			require.Equal(t, time.Duration(0), v)
		}
	}
	{
		backoff := 100 * time.Millisecond
		backoffs := ConstantBackoffs(5, backoff)
		require.Equal(t, 5, len(backoffs))
		for _, v := range backoffs {
			require.Equal(t, backoff, v)
		}
	}
	{
		backoff := 10 * time.Millisecond
		backoffs := ExponentialBackoffs(10, backoff)
		require.Equal(t, 10, len(backoffs))
		for i, v := range backoffs {
			require.Equal(t, backoff*(1<<uint(i)), v)
		}
	}
}
