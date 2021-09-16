package null

// Nullable interface wraps the "IsNull" function, which allows an
// object to identify if it is a null value or not.
type Nullable interface {

	// IsNull returns TRUE if the value of this object is null.  FALSE otherwise.
	IsNull() bool
}
