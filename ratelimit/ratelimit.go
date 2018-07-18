// Package ratelimit provides rate limiting implementations.
package ratelimit

import "context"

// RateLimiter is the abstraction for rate limiter.
type RateLimiter interface {
	// Take takes the size of available resources(maybe one or more tokens for TokenBucketRateLimiter),
	// wait until resources available or ctx canceled.
	Take(ctx context.Context, size int) error
	// Close closes the RateLimiter, after that, Take will block until ctx canceled.
	Close() error
}
