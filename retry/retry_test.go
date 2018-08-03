package retry

import (
	"runtime"
	"testing"
	"time"
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
		assert(t, true, err != nil)
		assert(t, cnt, 6)
		elapsed := time.Now().Sub(now)
		assert(t, elapsed > backoff*5, true)
		assert(t, elapsed < backoff*6, true)
	}
	{
		cnt := 0
		now := time.Now()
		err := Retry(ExponentialBackoffs(5, backoff), func() (State, error) {
			cnt++
			return Continue, nil
		})
		must(t, err)
		assert(t, cnt, 1)
		assert(t, true, time.Now().Sub(now) < backoff)
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
		assert(t, cnt, 2)
		assert(t, true, err == ErrNeedRetry)
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
		assert(t, cnt, 2)
		assert(t, true, err == nil)
	}
}

func TestBackoffFactory(t *testing.T) {
	{
		backoffs := ZeroBackoffs(3)
		assert(t, len(backoffs), 3)
		for _, v := range backoffs {
			assert(t, v, time.Duration(0))
		}
	}
	{
		backoff := 100 * time.Millisecond
		backoffs := ConstantBackoffs(5, backoff)
		assert(t, len(backoffs), 5)
		for _, v := range backoffs {
			assert(t, v, backoff)
		}
	}
	{
		backoff := 10 * time.Millisecond
		backoffs := ExponentialBackoffs(10, backoff)
		assert(t, len(backoffs), 10)
		for i, v := range backoffs {
			assert(t, v, backoff*(1<<uint(i)))
		}
	}
}

func assert(t *testing.T, actual interface{}, expect interface{}) {
	if actual != expect {
		_, fileName, line, _ := runtime.Caller(1)
		t.Fatalf("expect %v, got %v at (%v:%v)\n", expect, actual, fileName, line)
	}
}

func must(t *testing.T, err error) {
	if err != nil {
		_, fileName, line, _ := runtime.Caller(1)
		t.Fatalf("expect nil, got %v at (%v:%v)\n", err, fileName, line)
	}
}
