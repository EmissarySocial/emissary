package counter

import "github.com/benpate/rosetta/mapof"

// Counter tallies the number of times a key has been added
type Counter mapof.Int

// NewCounter returns a fully initialized Counter object.
// Counters are used to tally the number of times a key has been added.
func NewCounter() Counter {
	return make(Counter)
}

// Add increments the value of a key by 1
func (counter Counter) Add(key string) {
	counter[key]++
}

// Get returns the value of a key
func (counter Counter) Get(key string) int {
	return counter[key]
}
