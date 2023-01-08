package gofed

import (
	"context"
	"net/url"
)

// Lock lets your database implementation ensure asynchronous requests perform atomic
// changes to the underlying data. The id parameter acts as the primary key for the
// ActivityStreams entity that is going to be retrieved for either a read-only or
// read-write use-case.
//
// An implementation may decide to manage a dictionary of mutexes and lock a specific
// one, do nothing and instead rely on a particular database's transaction model, or
// something else.
func (db Database) Lock(c context.Context, id *url.URL) error {
	db.locks.Lock(id.String())
	return nil
}
