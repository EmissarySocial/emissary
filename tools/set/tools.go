package set

// Copy returns a new store that contains the same items as the given store.
func Copy[V Value, S Set[V]](from S, to S) {
	for value := range from.GetAll() {
		to.Put(value)
	}
}

// Intersect returns a new store that contains only the items that are in both stores.
func Intersect[V Value, S Set[V]](left S, right S, target S) {
	for value := range left.GetAll() {
		if value, ok := right.Get(value.ID()); ok {
			target.Put(value)
		}
	}
}

// Each calls the given function for each item in the set.
func Each[V Value, S Set[V]](set Set[V], f func(V)) {
	for value := range set.GetAll() {
		f(value)
	}
}

// Reduce performs a "reduce" operation on the given store.
func Reduce[V Value, S Set[V], R any](set S, f func(V, R) R) R {
	var result R
	for value := range set.GetAll() {
		result = f(value, result)
	}
	return result
}

// MaxKey returns the key of the item with the largest value.
func MaxKey[V Value, S Set[V]](set S) string {

	return Reduce(set, func(value V, key string) string {
		if value.ID() > key {
			return value.ID()
		}
		return key
	})
}
