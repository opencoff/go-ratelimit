// ratelimit_test.go -- Test harness for ratelimit
//
// (c) 2016 Sudhi Herle <sudhi@herle.net>
//
// Licensing Terms: GPLv2
//
// If you need a commercial license for this work, please contact
// the author.
//
// This software does not come with any express or implied
// warranty; it is provided "as is". No claim  is made to its
// suitability for any purpose.

package ratelimit // we use the same name to make it easy to test in _this_ dir

import (
	"fmt"
	"time"

	"runtime"
	"testing"
)

type tClock struct {
	time.Time
}

func newtClock() *tClock {
	t := &tClock{
		Time: time.Unix(0, 5000),
	}

	return t
}

func (f *tClock) Now() time.Time {
	return f.Time
}

func (f *tClock) Sleep(d time.Duration) {
	end := f.Time.Add(d)
	for f.Time.Before(end) {
		time.Sleep(200 * time.Microsecond)
	}
}

// Advance clock by 'by' milliseconds
func (f *tClock) advance(by int) {
	x := time.Duration(by) * time.Millisecond
	f.Time = f.Add(x)
}

// make an assert() function for use in environment 't' and return it
func newAsserter(t *testing.T) func(cond bool, msg string, args ...interface{}) {
	return func(cond bool, msg string, args ...interface{}) {
		if cond {
			return
		}

		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "???"
			line = 0
		}

		s := fmt.Sprintf(msg, args...)
		t.Fatalf("%s: %d: Assertion failed: %s\n", file, line, s)
	}
}

func TestUnlimited(t *testing.T) {
	assert := newAsserter(t)

	rl, err := New(0, 0)
	assert(err != nil, "expected err to be non-nil")

	rl, err = New(0, 1)
	assert(err == nil, "expected err to be nil; saw %s", err)

	assert(!rl.Limit(), "expected rl to not limit")
	assert(!rl.Limit(), "expected rl to not limit")
	assert(!rl.Limit(), "expected rl to not limit")
	assert(!rl.Limit(), "expected rl to not limit")
}

func TestLimit(t *testing.T) {
	clk := newtClock()
	assert := newAsserter(t)

	rl, err := NewWithClock(2, 1, clk)
	assert(err == nil, "expected err to be nil; saw %s", err)

	assert(!rl.Limit(), "expected rl to not limit")
	assert(!rl.Limit(), "expected rl to not limit")
	assert(rl.Limit(), "expected rl to limit")

	clk.advance(1000)
	assert(!rl.Limit(), "expected rl to not limit")
	assert(!rl.Limit(), "expected rl to not limit")
}

func TestLimitMany(t *testing.T) {
	clk := newtClock()
	assert := newAsserter(t)

	rl, err := NewWithClock(5, 2, clk)
	assert(err == nil, "expected err to be nil; saw %s", err)

	assert(!rl.Limit(), "expected rl to not limit")
	assert(!rl.Limit(), "expected rl to not limit")
	assert(!rl.Limit(), "expected rl to not limit")
	assert(!rl.Limit(), "expected rl to not limit")
	assert(!rl.Limit(), "expected rl to not limit")

	assert(rl.Limit(), "expected rl to limit")

	clk.advance(250)
	assert(rl.Limit(), "expected rl to limit")

	clk.advance(500)
	assert(!rl.Limit(), "expected rl to not limit")
	assert(rl.Limit(), "expected rl limit")
}

func TestBurst(t *testing.T) {
	clk := newtClock()
	assert := newAsserter(t)

	// 3 tokens every 2 secs with a burst of 5
	rl, err := NewBurstWithClock(3, 2, 5, clk)
	assert(err == nil, "expected err to be nil; saw %s", err)

	assert(rl.MaybeTake(5), "expected to take 5 burst")

	assert(rl.Limit(), "expected rl to limit after burst")

	clk.advance(700)
	assert(!rl.Limit(), "expected rl to not limit after refill")

	assert(!rl.MaybeTake(3), "expected burst 3 to fail")
}


func TestWait(t *testing.T) {
	clk := newtClock()
	assert := newAsserter(t)

	// 5 tokens per second
	rl, err := NewWithClock(5, 1, clk)
	assert(err == nil, "expected err to be nil; saw %s", err)

	assert(rl.MaybeTake(5), "expected to take 5")

	// If we ask to Wait(2), then we should return after we have
	// advanced the clock by the "cost"  = (1 * Nanosecond) / 5

	// cost per token (same formula as ratelimiter)
	cost := (1 * _NS) / 5
	want := 2 * cost

	exp := clk.Time.Add(time.Duration(want))

	// Ticker
	go func() {
		// wait for a sleeper to begin their sleep
		by := 10000 // microseconds
		for want > 0 {
			time.Sleep(10 * time.Microsecond)
			z := time.Duration(by) * time.Microsecond
			clk.Time = clk.Time.Add(z)
			want -= uint64(z)
		}
	}()

	assert(rl.Wait(2), "can't wait? %s", rl)

	delta := clk.Time.Sub(exp).Nanoseconds()
	assert(delta == 0, "clock drift? exp 0, saw %d", delta)
}

// vim: noexpandtab:ts=8:sw=8:tw=92:
