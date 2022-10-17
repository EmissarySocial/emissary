package set

// Value interface represents a keyed value that can be stored in a set
type Value interface {
	ID() string
}
