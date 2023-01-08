package activitypub

import "time"

type Clock struct{}

func NewClock() Clock {
	return Clock{}
}

func (service Clock) Now() time.Time {
	return time.Now()
}
