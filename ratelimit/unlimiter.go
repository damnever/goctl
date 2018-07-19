package ratelimit

import "context"

type unlimiter struct{}

// NewUnlimiter creates an unlimiter, use it with caution.
func NewUnlimiter() RateLimiter {
	return &unlimiter{}
}

func (l *unlimiter) Take(context.Context, int) error {
	return nil
}

func (l *unlimiter) Close() error {
	return nil
}
