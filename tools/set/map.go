package set

import (
	"github.com/benpate/derp"
	"golang.org/x/exp/constraints"
)

// Map is a simple in-memory map-based set for arbitrary data.
type Map[K constraints.Ordered, V Value[K]] map[K]V

// NewMap returns a new Map that is populated with the given items.
func NewMap[K constraints.Ordered, V Value[K]](values ...V) Map[K, V] {
	result := make(Map[K, V], 0)

	for _, value := range values {
		result.Put(value)
	}

	return result
}

// Len returns the number of items in the set.
func (set Map[K, V]) Len() int {
	return len(set)
}

// Get returns the object with the given ID.  If no object with the given ID is found, Get returns an empty object.
func (set Map[K, V]) Get(key K) (V, error) {

	if value, ok := set[key]; ok {
		return value, nil
	}

	var result V

	return result, derp.NewNotFoundError("set.Map.Get", "ID not found", key)
}

// GetAll returns a channel that will yield all of the items in the set.
func (set Map[S, T]) GetAll() <-chan T {
	ch := make(chan T)
	go func() {
		for _, v := range set {
			ch <- v
		}
		close(ch)
	}()
	return ch
}

// Put adds/updates an item in the set.
func (set Map[K, V]) Put(value V) {
	set[value.ID()] = value
}

// Delete removes an item from the set.
func (set Map[K, V]) Delete(key K) {
	delete(set, key)
}
