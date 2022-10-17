package set

// Set interface defines the functions that a set must implement
type Set[V Value] interface {
	Len() int
	Keys() []string
	Get(key string) (V, bool)
	GetAll() <-chan V
	Put(value V)
	Delete(key string)
}
