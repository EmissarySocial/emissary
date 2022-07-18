package set

import "golang.org/x/exp/constraints"

// Copy returns a new store that contains the same items as the given store.
func Copy[K constraints.Ordered, V Value[K], S Set[K, V]](from S, to S) {
	for value := range from.GetAll() {
		to.Put(value)
	}
}

// Intersect returns a new store that contains only the items that are in both stores.
func Intersect[K constraints.Ordered, V Value[K], S Set[K, V]](left S, right S, target S) {
	for value := range left.GetAll() {
		if value, err := right.Get(value.ID()); err == nil {
			target.Put(value)
		}
	}
}

// Each calls the given function for each item in the set.
func Each[K constraints.Ordered, V Value[K], S Set[K, V]](set Set[K, V], f func(V)) {
	for value := range set.GetAll() {
		f(value)
	}
}

// Reduce performs a "reduce" operation on the given store.
func Reduce[K constraints.Ordered, V Value[K], S Set[K, V], R any](set S, f func(V, R) R) R {
	var result R
	for value := range set.GetAll() {
		result = f(value, result)
	}
	return result
}

// MaxKey returns the key of the item with the largest value.
func MaxKey[K constraints.Ordered, V Value[K], S Set[K, V]](set S) K {

	return Reduce[K](set, func(value V, key K) K {
		if value.ID() > key {
			return value.ID()
		}
		return key
	})
}
