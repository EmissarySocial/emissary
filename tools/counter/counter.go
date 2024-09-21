package counter

import "github.com/benpate/rosetta/mapof"

type Counter mapof.Int

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
