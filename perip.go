// perip.go -- per IP:port rate limiter
//
// (c) 2013 Sudhi Herle <sudhi-dot-herle-at-gmail-com>
//
// License: GPLv2
//

// This uses the underlying Ratelimiter class to provide a simple
// interface for managing rate limits per source (IP:port)
package ratelimit

import (
	"net"
	"sync"
)

// Manages a map of source IP:port to underlying ratelimiter.
// In case of a DoS/DDoS attack, this map can grow unbounded.
// TODO Add some kind of limit to the # of map entries.
type PerIPRatelimiter struct {
	rl map[string]*Ratelimiter
	mu sync.Mutex

	rate, per int // rate/per for the underlying limiter
}

func NewPerIPRatelimiter(ratex, perx int) (*PerIPRatelimiter, error) {

	p := &PerIPRatelimiter{
		rl:   make(map[string]*Ratelimiter),
		rate: ratex,
		per:  perx,
	}

	return p, nil
}

// Return true if the source 'a' needs to be rate limited, false
// otherwise.
func (p *PerIPRatelimiter) Limit(a net.Addr) bool {
	s := a.String()

	p.mu.Lock()

	r, ok := p.rl[s]
	if !ok {
		r, _ = New(p.rate, p.per)
		p.rl[s] = r
	}

	p.mu.Unlock()

	return r.Limit()
}

// vim: noexpandtab:ts=8:sw=8:tw=92:
