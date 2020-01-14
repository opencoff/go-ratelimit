# go-ratelimit - Simple wrapper around `golang.org/x/time/rate`

## What is it?
Token bucket ratelimiter for golang; it wraps the `Limiter` in
`golang.org/x/time/rate`. It implements global *and* per-host rate limits.
It uses an LRU cache to cache the most frequently used per-host limiters.

