package gofed

import (
	"context"

	"github.com/go-fed/activity/streams/vocab"
)

// Update is the same as Create except it is expected that the object already is in
// the database. The entity with the same id should be overwritten by the provided
// value. You do not need to worry about the ActivityPub specification talking about
// whether an Update means a partial-update or complete-replacement, as the library
// has already done this for you, so it is safe to simply replace the row.
func (db Database) Update(c context.Context, asType vocab.Type) error {
	return nil
}
