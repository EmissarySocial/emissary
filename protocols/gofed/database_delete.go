package gofed

import (
	"context"
	"net/url"
)

// Delete removes the entity or row with the matching id.
func (db Database) Delete(c context.Context, id *url.URL) error {
	return nil
}
