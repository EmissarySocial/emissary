package iterators

import "github.com/benpate/data"

func Map[A any, B any](iterator data.Iterator, constructor func() A, mapper func(A) B) []B {

	result := make([]B, 0, iterator.Count())

	value := constructor()

	for iterator.Next(&value) {
		result = append(result, mapper(value))
		value = constructor()
	}

	return result
}
