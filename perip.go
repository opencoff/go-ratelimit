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
	"sync"

	"github.com/opencoff/golang-lru"
)

// Manages a map of source IP:port to underlying ratelimiter.
// Note: In case of a DoS/DDoS attack, this map can grow unbounded.
// TODO Add some kind of limit to the # of map entries.
type PerIPRateLimiter struct {
	sync.Mutex

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

// Return true if the source 'a' needs to be rate limited, false
// otherwise.
func (p *PerIPRateLimiter) Limit(a net.Addr) bool {

	// Unlimited rate
	if p.rl == nil {
		return false
	}

	s := a.String()

	var z *RateLimiter

	// XXX If only we had an LRU "Probe" method that inserted if non-existent and
	// returned an existing entry if it did.

	p.Lock()
	if r, ok := p.rl.Get(s); ok {
		z = r.(*RateLimiter)
	} else {
		z, _ = New(p.rate, p.per)
		p.rl.Add(s, z)
	}
	p.Unlock()

	return z.Limit()
}

// vim: noexpandtab:ts=8:sw=8:tw=92:
