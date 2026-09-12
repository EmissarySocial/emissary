// Package set contains some nifty tools for manipulating sets of data,
// including implementations for slices AND maps.  This package should probably
// make its way into rosetta, once things settle down.
package set

// Set defines the functions that a set must implement
//
// TODO: Should we migrate the `set` package to Rosetta? Interop with other libs, like `sliceof` and `mapof`?
type Set[V Value] interface {
	Len() int
	Keys() []string
	Get(key string) (V, bool)
	GetAll() <-chan V
	Put(value V)
	Delete(key string)
}
