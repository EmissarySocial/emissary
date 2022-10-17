package set

// Map is a simple in-memory map-based set for arbitrary data.
type Map[V Value] map[string]V

// NewMap returns a new Map that is populated with the given items.
func NewMap[V Value](values ...V) Map[V] {
	result := make(Map[V], 0)

	for _, value := range values {
		result.Put(value)
	}

	return result
}

// Len returns the number of items in the set.
func (set Map[V]) Len() int {
	return len(set)
}

// Keys returns a list of all the keys in the set.
func (set Map[V]) Keys() []string {
	result := make([]string, len(set))

	index := 0
	for key := range set {
		result[index] = key
		index++
	}

	return result
}

// Get returns the object with the given ID.  If no object with the given ID is found, Get returns an empty object.
func (set Map[V]) Get(key string) (V, bool) {

	if value, ok := set[key]; ok {
		return value, true
	}

	var result V

	return result, false
}

// GetAll returns a channel that will yield all of the items in the set.
func (set Map[V]) GetAll() <-chan V {
	ch := make(chan V)
	go func() {
		for _, v := range set {
			ch <- v
		}
		close(ch)
	}()
	return ch
}

// Put adds/updates an item in the set.
func (set Map[V]) Put(value V) {
	set[value.ID()] = value
}

// Delete removes an item from the set.
func (set Map[V]) Delete(key string) {
	delete(set, key)
}
