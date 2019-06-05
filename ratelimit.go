// ratelimit.go - Token bucket ratelimiter
//
// (c) 2013 Sudhi Herle <sudhi-dot-herle-at-gmail-com>
//
// License: GPLv2
//

// Package ratelimit implements a token bucket rate limiter. It does NOT use any
// timers or channels in its implementation. The core idea is that every call
// to ask for a Token also "drip fills" the bucket with fractional tokens.
// To evenly drip-fill the bucket, we do all our calculations in nanoseconds.
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
	mu     sync.Mutex
	cost   uint64    // cost per packet in units of nanoseconds
	tokens uint64    // tokens in the bucket
	maxtok uint64    // max tokens possible in the given time interval
	last   time.Time // last time we refreshed tokens

	rate  uint64 // rate of drain
	burst uint64 // burst rate
	per   uint64 // seconds over which rate is measured

	clock Clock // timekeeper
}

// Clock provides an interface to timekeeping. It is used in test harness.
type Clock interface {
	// Return current time in seconds
	Now() time.Time
}

const (
	_NS uint64 = 1000000000 // number of nanosecs in 1 sec
)

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
		return nil, fmt.Errorf("ratelimit: duration can't be zero")
	}

	if burst == 0 {
		burst = rate
	}

	r := &RateLimiter{
		rate:  uint64(rate),
		burst: uint64(burst),
		per:   uint64(per),

		last:  clk.Now(),
		clock: clk,
	}

	if rate > 0 {
		r.cost = (r.per * _NS) / r.rate
		r.tokens = r.burst * r.cost
		r.maxtok = r.tokens
	}

	return r, nil
}

// Reset the rate limiter
func (r *RateLimiter) Reset() {
	if r.rate > 0 {
		r.tokens = r.burst * r.cost
		r.last = r.clock.Now()
	}
}

// MaybeTake attempts to take 'vn' tokens from the rate limiter.
// Returns true if it can take all of them, false otherwise.
func (r *RateLimiter) MaybeTake(vn uint) (ok bool) {
	if r.rate == 0 {
		return true
	}

	now := r.clock.Now()

	r.mu.Lock()
	r.tokens += uint64(now.Sub(r.last).Nanoseconds())
	r.last = now

	if r.tokens > r.maxtok {
		r.tokens = r.maxtok
	}

	if want := uint64(vn) * r.cost; r.tokens >= want {
		r.tokens -= want
		ok = true
	}

	r.mu.Unlock()
	return ok
}

// Return true if the current call exceeds the set rate, false
// otherwise
func (r *RateLimiter) Limit() bool {
	return !r.MaybeTake(1)
}

// Stringer implementation for RateLimiter
func (r RateLimiter) String() string {
	return fmt.Sprintf("ratelimiter(%d/%ds +%d): cost/pkt: %d toks, max %d toks; avail %d toks",
		r.rate, r.per, r.burst, r.cost, r.maxtok, r.tokens)
}

// vim: noexpandtab:ts=8:sw=8:tw=92:
