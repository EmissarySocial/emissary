package data

// Session represents a single database session, that is opened to support a single transactional request, and then closed
// when this transaction is complete
type Session interface {
	Collection(collection string) Collection
	Close()
}
