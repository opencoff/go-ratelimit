//
// Ratelimiting incoming connections - Small Library
//
// (c) 2013 Sudhi Herle <sudhi-dot-herle-at-gmail-com>
//
// License: GPLv2
//

// Notes:
//  - This is a very simple interface for token-bucket rate limiter.
//  - Based on Anti Huimaa's very clever token bucket algorithm:
//    http://stackoverflow.com/questions/667508/whats-a-good-rate-limiting-algorithm
//  - The core idea is that every call to ask for a Token also "drip fills"
//    the bucket with fractional tokens.
//  - To evenly drip-fill the bucket, we do all our calculations in
//    millseconds
//  - A Clock interface abstracts the timekeeping; useful for test harness
//  - Most callers will call the New() function; the test suite
//    calls the NewWithClock() constructor.
//
// Usage:
//    rate = 1000
//    per  = 5
//    rl = ratelimit.New(rate, per) // ratelimit to 1000 every 5 seconds
//
//    ....
//    if rl.Limit() {
//       drop_connection(conn)
//    }
//
package ratelimit

import (
	"time"
)

type Ratelimiter struct {
	rate      float64   // rate of drain
	per       float64   // seconds over which the rate is measured
	frac      float64   // rate / milliseconds; this is the drip fill rate
	tokens    float64   // tokens in the bucket
	last      time.Time // last time we refreshed the tokens
	unlimited bool      // set to true if rate is 0
	clock     Clock     // timekeeper
}

// Clock provides an interface to timekeeping. It was created to help with tests.
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
func New(rate, per int) (*Ratelimiter, error) {

	clk := defaultTime(0)
	return NewWithClock(rate, per, &clk)
}

// Make a new rate limiter using a custom timekeeper
func NewWithClock(rate, per int, clk Clock) (*Ratelimiter, error) {

	if rate <= 0 {
		rate = 0
	}
	if per <= 0 {
		per = 1
	}

	r := Ratelimiter{
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
func (r *Ratelimiter) elapsed(last time.Time) (now time.Time, since float64) {
	now = r.clock.Now()
	nsec := now.Sub(last).Nanoseconds()
	since = float64(nsec) / 1.0e6

	return now, since
}

// Return true if the current call exceeds the set rate, false
// otherwise
func (r *Ratelimiter) Limit() bool {

	var since float64

	// handle cases where rate in config file is unset - defaulting
	// to unlimited
	if r.unlimited {
		return false
	}

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
