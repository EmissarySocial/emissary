package gofed

import (
	"context"
	"net/url"
)

// Unlock is the counterpart to the Lock method. The id parameter acts as the primary
// key for the ActivityStreams entity that has already been retrieved, and possibly
// stored, for either a read-only or read-write use-case.
func (db Database) Unlock(c context.Context, id *url.URL) error {
	db.locks.Unlock(id.String())
	return nil
}
