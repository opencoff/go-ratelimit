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
	"fmt"
	"sync"
	"time"
)

// RateLimiter represents a token-bucket rate limiter
type RateLimiter struct {
	rate      float64   // rate of drain
	per       float64   // seconds over which the rate is measured
	frac      float64   // rate / milliseconds; this is the drip fill rate
	tokens    float64   // tokens in the bucket
	clamp     float64   // max tokens we will ever have
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
func New(rate, per uint) (*RateLimiter, error) {
	clk := defaultTime(0)
	return NewBurstWithClock(rate, per, 0, &clk)
}

// Make a new rate limiter using a custom timekeeper
func NewWithClock(rate, per uint, clk Clock) (*RateLimiter, error) {
	return NewBurstWithClock(rate, per, 0, clk)
}

// Create new limiter that limits to 'rate' every 'per' seconds with
// burst of 'b' tokens in the same time period. The notion of burst
// is only meaningful when it is larger than its normal rate. Thus,
// bursts smaller than the actual rate are ignored.
func NewBurst(rate, per, burst uint) (*RateLimiter, error) {
	clk := defaultTime(0)

	return NewBurstWithClock(rate, per, burst, &clk)
}

// Create new limiter with a custom time keeper that limits to 'rate'
// every 'per' seconds with burst of 'b' tokens in the same time period.
// The notion of burst is only meaningful when it is larger than its
// normal rate. Thus, bursts smaller than the actual rate are ignored.
func NewBurstWithClock(rate, per, burst uint, clk Clock) (*RateLimiter, error) {
	if per == 0 {
		per = 1
	}

	// burst is only meaningful if it is larger than the rate. thus,
	// the max number of tokens when we dripfill is the larger of the
	// rate or the burst.

	clamp := rate
	if burst > rate {
		clamp = burst
	}

	r := RateLimiter{
		rate:   float64(rate),
		frac:   float64(rate) / float64(per*1000),
		last:   clk.Now(),
		tokens: float64(clamp),
		clamp:  float64(clamp),
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

// Return true if we can take 'n' tokens, false otherwise
func (r *RateLimiter) CanTake(vn uint) bool {

	if r.unlimited {
		return true
	}

	r.Lock()
	defer r.Unlock()

	var since float64

	n := float64(vn)
	r.last, since = r.elapsed(r.last)
	r.tokens += (since * r.frac)

	if r.tokens > r.clamp {
		r.tokens = r.clamp
	}

	if r.tokens < n {
		return false
	}

	r.tokens -= n
	return true
}

// Return true if the current call exceeds the set rate, false
// otherwise
func (r *RateLimiter) Limit() bool {
	return !r.CanTake(1)
}


// Stringer implementation for RateLimiter
func (r RateLimiter) String() string {
	return fmt.Sprintf("dripfill: %3.4f toks every ms burst: %3.1f toks; %3.4f toks avail",
		r.frac, r.clamp, r.tokens)
}

// vim: noexpandtab:ts=8:sw=8:tw=92:
