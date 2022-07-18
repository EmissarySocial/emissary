package set

import (
	"sort"

	"github.com/benpate/derp"
	"golang.org/x/exp/constraints"
)

// Slice is a simple in-memory slice-based set for arbitrary data.
type Slice[K constraints.Ordered, V Value[K]] []V

// NewSlice returns a new Slice that is populated with the given items.
func NewSlice[K constraints.Ordered, V Value[K]](values ...V) Slice[K, V] {
	result := make(Slice[K, V], 0)

	for _, value := range values {
		result.Put(value)
	}

	return result
}

// Len returns the number of items in the set.
func (set Slice[K, V]) Len() int {
	return len(set)
}

// Get returns the object with the given ID.  If no object with the given ID is found, Get returns an empty object.
func (set Slice[K, V]) Get(key K) (V, error) {

	for _, value := range set {
		if value.ID() == key {
			return value, nil
		}
	}

	var value V
	return value, derp.NewNotFoundError("store.Slice.Get", "ID not found", key)
}

// GetAll returns a channel that will yield all of the items in the store.
func (set Slice[K, V]) GetAll() <-chan V {
	ch := make(chan V)
	go func() {
		for _, value := range set {
			ch <- value
		}
		close(ch)
	}()
	return ch
}

// Put adds/updates an item in the set.
func (set *Slice[K, V]) Put(value V) {

	// Try to find the item in the set
	for index, item := range *set {
		if item.ID() == value.ID() {
			(*set)[index] = value
			return
		}
	}

	// Fall through means that the item was not found in the set. So let's append it.
	*set = append(*set, value)
}

// Delete removes an item from the set.
func (set *Slice[K, V]) Delete(key K) {
	for index, value := range *set {
		if value.ID() == key {
			*set = append((*set)[:index], (*set)[index+1:]...)
			return
		}
	}
}

func (set Slice[K, V]) Less(i int, j int) bool {
	return set[i].ID() < set[j].ID()
}

func (set *Slice[K, V]) Swap(i int, j int) {
	(*set)[i], (*set)[j] = (*set)[j], (*set)[i]
}

func (set *Slice[K, V]) Sort() {
	sort.Sort(set)
}
