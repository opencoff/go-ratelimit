// ratelimit.go - Token bucket ratelimiter
//
// (c) 2013 Sudhi Herle <sudhi-dot-herle-at-gmail-com>
//
// License: GPLv2
//

// Package ratelimit implements a token bucket rate limiter. It does NOT use any
// timers or channels in its implementation. The core idea is that every call
// to ask for a Token also "drip fills" the bucket with fractional tokens.
// To evenly drip-fill the bucket, we do all our calculations in millseconds.
//
// To ratelimit incoming connections on a per source basis, a convenient helper
// constructor is available for "PerIPRateLimiter".
//
// Usage:
//    // Ratelimit to 1000 every 5 seconds
//    rl = ratelimit.New(1000, 5)
//
//    ....
//    if rl.Limit() {
//       dropConnection(conn)
//    }
package ratelimit

// Notes:
//
// - A Clock interface abstracts the timekeeping; useful for test harness
// - Most callers will call the New() function; the test suite
//   calls the NewWithClock() constructor.
// - This is a very simple interface for token-bucket rate limiter.
// - Based on Anti Huimaa's very clever token bucket algorithm:
//   http://stackoverflow.com/questions/667508/whats-a-good-rate-limiting-algorithm
//

import (
	"sync"
	"time"
)

// RateLimiter represents a token-bucket rate limiter
type RateLimiter struct {
	rate      float64   // rate of drain
	per       float64   // seconds over which the rate is measured
	frac      float64   // rate / milliseconds; this is the drip fill rate
	tokens    float64   // tokens in the bucket
	last      time.Time // last time we refreshed the tokens
	unlimited bool      // set to true if rate is 0
	clock     Clock     // timekeeper

	sync.Mutex
}

// Clock provides an interface to timekeeping. It is used in test harness.
type Clock interface {
	// Return current time in seconds
	Now() time.Time
}

// dummy type
type defaultTime int

func (*defaultTime) Now() time.Time {
	return time.Now()
}

// Create new limiter that limits to 'rate' every 'per' seconds
func New(rate, per int) (*RateLimiter, error) {

	clk := defaultTime(0)
	return NewWithClock(rate, per, &clk)
}

// Make a new rate limiter using a custom timekeeper
func NewWithClock(rate, per int, clk Clock) (*RateLimiter, error) {

	if rate <= 0 {
		rate = 0
	}
	if per <= 0 {
		per = 1
	}

	r := RateLimiter{
		rate:   float64(rate),
		frac:   float64(rate) / float64(per*1000),
		last:   clk.Now(),
		tokens: float64(rate),
		clock:  clk,
	}

	if rate == 0 {
		r.unlimited = true
	}

	return &r, nil
}

// Compute elapsed time in milliseconds since 'last'
func (r *RateLimiter) elapsed(last time.Time) (now time.Time, since float64) {
	now = r.clock.Now()
	nsec := now.Sub(last).Nanoseconds()
	since = float64(nsec) / 1.0e6

	return now, since
}

// Return true if the current call exceeds the set rate, false
// otherwise
func (r *RateLimiter) Limit() bool {

	// handle cases where rate in config file is unset - defaulting
	// to unlimited
	if r.unlimited {
		return false
	}

	r.Lock()
	defer r.Unlock()

	var since float64

	r.last, since = r.elapsed(r.last)
	r.tokens += since * r.frac

	// Clamp number of tokens in the bucket. Don't let it get
	// unboundedly large
	if r.tokens > r.rate {
		r.tokens = r.rate
	}

	if r.tokens < 1.0 {
		return true
	}

	r.tokens -= 1.0
	return false
}

// vim: noexpandtab:ts=8:sw=8:tw=92:
