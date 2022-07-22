package set

import "golang.org/x/exp/constraints"

// Set interface defines the functions that a set must implement
type Set[K constraints.Ordered, V Value[K]] interface {
	Len() int
	Get(key K) (V, error)
	GetAll() <-chan V
	Put(value V)
	Delete(key K)
}

// Value interface represents a keyed value that can be stored in a set
type Value[K constraints.Ordered] interface {
	ID() K
}
