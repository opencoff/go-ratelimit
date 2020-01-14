// perip.go -- per IP:port rate limiter
//
// (c) 2013 Sudhi Herle <sudhi-dot-herle-at-gmail-com>
//
// License: GPLv2
//

package ratelimit

import (
	"fmt"
	"net"
	"github.com/opencoff/golang-lru"
)

// Manages a map of source IP:port to underlying ratelimiter
// Each entry is in a LRU Cache. The Per-IP Limiter is bounded to a
// maximum size when it is constructed.
type PerIPRateLimiter struct {
	rl *lru.TwoQueueCache

	rate, per uint // rate/per for the underlying limiter
}

// Create a new per-source rate limiter to limit each IP (host)
// to 'ratex' every 'perx' seconds. Hold a maximum of 'max'
// IP addresses in the rate-limiter
func NewPerIP(ratex, perx uint, max int) (*PerIPRateLimiter, error) {
	var err error

	// Validate params one time.
	_, err = New(ratex, perx)
	if err != nil {
		return nil, err
	}

	var q *lru.TwoQueueCache

	if max <= 0 {
		return nil, fmt.Errorf("per-ip rate limiter needs a non-zero max size (saw %d)", max)
	}

	if ratex > 0 {
		// XXX hard-coded ratios?
		q, err = lru.New2QParams(max, 0.65, 0.35)
		if err != nil {
			return nil, err
		}
	}

	p := &PerIPRateLimiter{
		rl:   q,
		rate: ratex,
		per:  perx,
	}

	return p, nil
}

func (p *PerIPRateLimiter) probe(a net.Addr) *RateLimiter {
	s := a.String()
	v, _ := p.rl.Probe(s, func (_ interface{}) interface{} {
		z, _ := New(p.rate, p.per)
		return z
	})
	return v.(*RateLimiter)
}

// Reset ratelimiter state for this host
func (p *PerIPRateLimiter) Reset(a net.Addr) {
	z := p.probe(a)
	z.Reset()
}

// MaybeTake attempts to take 'n' tokens for the host 'a'
// Returns true if it can take all of them, false otherwise.
func (p *PerIPRateLimiter) MaybeTake(a net.Addr, n uint) bool {
	// Unlimited rate
	if p.rl == nil {
		return false
	}
	z := p.probe(a)
	return z.MaybeTake(n)
}

// Return true if the source 'a' needs to be rate limited, false
// otherwise.
func (p *PerIPRateLimiter) Limit(a net.Addr) bool {
	// Unlimited rate
	if p.rl == nil {
		return false
	}

	z := p.probe(a)
	return z.Limit()
}


// Wait until requested tokens (n) become available for this host 'a'
func (p *PerIPRateLimiter) Wait(a net.Addr, n uint) bool {
	// Unlimited rate
	if p.rl == nil {
		return false
	}

	z := p.probe(a)
	return z.Wait(n)
}

// vim: noexpandtab:ts=8:sw=8:tw=92:
