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
	// module under test
	//"github.com/sign"
)

type tClock struct {
	time.Time
}

func newtClock() *tClock {
	t := &tClock{Time: time.Unix(789133, 899152383)}
	return t
}

func (f *tClock) Now() time.Time {
	return f.Time
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

// vim: noexpandtab:ts=8:sw=8:tw=92:
