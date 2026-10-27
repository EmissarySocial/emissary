package handler

import (
	"context"

	"github.com/benpate/data"
)

// unusedSession is a data.Session for a test whose code path must never reach the database.
// Every method panics, so reaching the database fails the test instead of passing it.
type unusedSession struct{}

// Collection implements the data.Session interface, and panics
func (unusedSession) Collection(name string) data.Collection {
	panic("handler test: this path must not reach the database, but it opened " + name)
}

// Context implements the data.Session interface, and panics
func (unusedSession) Context() context.Context {
	panic("handler test: this path must not reach the database, but it read the session context")
}

// Close implements the data.Session interface, and panics
func (unusedSession) Close() {
	panic("handler test: this path must not reach the database, but it closed the session")
}
