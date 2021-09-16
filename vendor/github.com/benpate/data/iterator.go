package data

// Iterator interface allows callers to iterator over a large number of items in an array/slice
type Iterator interface {
	Next(Object) bool
	Count() int
	Close() error
}
