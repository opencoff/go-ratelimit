// ratelimit.go - Rate limiter wrapper around golang.org/x/time/rate
//
// Author: Sudhi Herle <sudhi@herle.net>
// License: GPLv2
//
// This software does not come with any express or implied
// warranty; it is provided "as is". No claim  is made to its
// suitability for any purpose.
//

// Package ratelimit wraps the Limiter from golang.org/x/time/rate
// and creates a simple interface for global and per-host limits.
//
// Usage:
//
//	// Ratelimit globally to 1000 req/s, per-host to 5 req/s and cache
//	// latest 30000 per-host limits
//	rl = ratelimit.New(1000, 5, 30000)
//
//	....
//	if !rl.Allow() {
//	   dropConnection(conn)
//	}
//
//	if  !rl.AllowHost(conn.RemoteAddr()) {
//	   dropConnection(conn)
//	}
package ratelimit

import (
	"context"
	"fmt"
	"github.com/hashicorp/golang-lru/v2"
	"golang.org/x/time/rate"
	"net"
)

// Limiter controls how frequently events are allowed to happen globally or
// per-host. It uses a token-bucket limiter for the global limit and instantiates
// a token-bucket limiter for every unique host. The number of per-host limiters
// is limited to an upper bound ("cache size").
//
// A negative rate limit means "no limit" and a zero rate limit means "Infinite".
type Limiter struct {
	// Global rate limiter; thread-safe
	gl *rate.Limiter

	// Per-host limiter organized as an LRU cache; thread-safe
	h *lru.TwoQueueCache[string, *rate.Limiter]

	// per host rate limit (qps)
	p rate.Limit
	g rate.Limit

	// burst rate for per-host
	b int

	cache int
}

// Create a new token bucket rate limiter that limits globally at 'g'  requests/sec
// and per-host at 'p' requests/sec; It remembers the rate of the 'cachesize' most
// recent hosts (and their limits). The burst rates are pre-configured to be:
// Global burst limit: 3 * b; Per host burst limit:  2 * p
func New(g, p, cachesize int) (*Limiter, error) {
	l, err := lru.New2Q[string, *rate.Limiter](cachesize)
	if err != nil {
		return nil, fmt.Errorf("ratelimit: can't create LRU cache: %s", err)
	}

	b := 2 * p
	if b < 0 {
		b = 0
	}

	gl := limit(g)
	pl := limit(p)

	r := &Limiter{
		gl:    rate.NewLimiter(gl, 3*g),
		h:     l,
		p:     pl,
		g:     gl,
		b:     b,
		cache: cachesize,
	}

	return r, nil
}

// Wait blocks until the ratelimiter permits the configured global rate limit.
// It returns an error if the burst exceeds the configured limit or the
// context is cancelled.
func (r *Limiter) Wait(ctx context.Context) error {
	return r.gl.Wait(ctx)
}

// WaitHost blocks until the ratelimiter permits the configured per-host
// rate limit from host 'a'.
// It returns an error if the burst exceeds the configured limit or the
// context is cancelled.
func (r *Limiter) WaitHost(ctx context.Context, a net.Addr) error {
	rl := r.getRL(a)
	return rl.Wait(ctx)
}

// Allow returns true if the global rate limit can consume 1 token and
// false otherwise. Use this if you intend to drop/skip events that exceed
// a configured global rate limit, otherwise, use Wait().
func (r *Limiter) Allow() bool {
	return r.gl.Allow()
}

// AllowHost returns true if the per-host rate limit for host 'a' can consume
// 1 token and false otherwise. Use this if you intend to drop/skip events
// that exceed a configured global rate limit, otherwise, use WaitHost().
func (r *Limiter) AllowHost(a net.Addr) bool {
	rl := r.getRL(a)
	return rl.Allow()
}

// String returns a printable representation of the limiter
func (r Limiter) String() string {
	return fmt.Sprintf("ratelimiter: Global %4.2 rps, Per host %4.2 rps, LRU cache %d entries",
		r.g, r.p, r.cache)
}

// get or create a new per-host rate limiter.
// this function evicts the least used limiter from the LRU cache
func (r *Limiter) getRL(a net.Addr) *rate.Limiter {
	k := host(a)
	rl, ok := r.h.Get(k)
	if !ok {
		rl = rate.NewLimiter(r.p, r.b)
		r.h.Add(k, rl)
	}
	return rl
}

// return the host from the address
func host(a net.Addr) string {
	s := a.String()
	if h, _, err := net.SplitHostPort(s); err == nil {
		return h
	}
	return s
}

func limit(r int) rate.Limit {
	var g rate.Limit

	switch {
	case r < 0:
		g = rate.Inf
	case r == 0:
		g = 0.0
	default:
		g = rate.Limit(r)
	}

	return g
}

// EOF
