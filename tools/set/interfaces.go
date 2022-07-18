package set

import "golang.org/x/exp/constraints"

type Value[K constraints.Ordered] interface {
	ID() K
}

type Set[K constraints.Ordered, V Value[K]] interface {
	Len() int
	Get(key K) (V, error)
	GetAll() <-chan V
	Put(value V)
	Delete(key K)
}
