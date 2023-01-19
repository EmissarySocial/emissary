package set

import (
	"sort"

	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
)

// Slice is a simple in-memory slice-based set for arbitrary data.
type Slice[V Value] sliceof.Object[V]

// NewSlice returns a new Slice that is populated with the given items.
func NewSlice[V Value](values ...V) Slice[V] {
	result := make(Slice[V], 0)

	for _, value := range values {
		result.Put(value)
	}

	return result
}

// Len returns the number of items in the set.
func (set Slice[V]) Len() int {
	return len(set)
}

// Keys returns a list of all the keys in the set.
func (set Slice[V]) Keys() []string {
	result := make([]string, len(set))

	for index, value := range set {
		result[index] = value.ID()
	}

	return result
}

// Get returns the object with the given ID.  If no object with the given ID is found, Get returns an empty object.
func (set Slice[V]) Get(key string) (V, bool) {

	for _, value := range set {
		if value.ID() == key {
			return value, true
		}
	}

	var value V
	return value, false
}

// GetAll returns a channel that will yield all of the items in the store.
func (set Slice[V]) GetAll() <-chan V {
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
func (set *Slice[V]) Put(value V) {

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
func (set *Slice[V]) Delete(key string) {
	for index, value := range *set {
		if value.ID() == key {
			*set = append((*set)[:index], (*set)[index+1:]...)
			return
		}
	}
}

func (set Slice[V]) Less(i int, j int) bool {
	return set[i].ID() < set[j].ID()
}

func (set *Slice[V]) Swap(i int, j int) {
	(*set)[i], (*set)[j] = (*set)[j], (*set)[i]
}

func (set *Slice[V]) Sort() {
	sort.Sort(set)
}

/******************************************
 * schema Interfaces
 ******************************************/

func (set *Slice[V]) GetObjectOK(name string) (any, bool) {

	if index, ok := schema.Index(name); ok {

		for index >= len(*set) {
			var newItem V
			*set = append(*set, newItem)
		}

		return &(*set)[index], true
	}

	return nil, false
}

func (set Slice[V]) Length() int {
	return len(set)
}

func (set *Slice[V]) Remove(name string) {
	set.Delete(name)
}
