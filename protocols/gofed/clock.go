package gofed

import "time"

// Clock implements the pub.Clock interface, which is a simple wrapper
// for the Now() method.  This prevents dependency on the time package
// which is useful for testing, but not production.
type Clock struct{}

func NewClock() Clock {
	return Clock{}
}

func (clock Clock) Now() time.Time {
	return time.Now()
}
