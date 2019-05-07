# go-ratelimit - Simple token bucket rate limiter

## What is it?
Token bucket ratelimiter for golang; this implementation doesn't use
any timers or channels.

- The core idea is that every call to ask for a token also "drip fills"
  the bucket with fractional tokens.
- To evenly drip-fill the bucket, we do all our calculations in
  millseconds.
- The interface supports creating rate limiter instances with a burst factor.

There is an ancilliary class to do per-IP ratelimiting that uses the
underlying library. The per-IP module use a fixed size LRU cache of the underlying
ratelimiter.

## Notes
This is based on Anti Huimaa's very clever token bucket algorithm:
http://stackoverflow.com/questions/667508/whats-a-good-rate-limiting-algorithm
