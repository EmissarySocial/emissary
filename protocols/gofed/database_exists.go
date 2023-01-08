package gofed

import (
	"context"
	"net/url"
)

// Exists returns TRUE if the database has an entity or row with the id
func (db Database) Exists(c context.Context, id *url.URL) (exists bool, err error) {
	return false, nil
}
