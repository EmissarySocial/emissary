package db

import (
	"context"
	"net/url"
)

func (db *Database) Owns(ctx context.Context, itemURL *url.URL) (owns bool, err error) {
	// Owns just determines if the ActivityPub id is owned by this server.
	// TODO: HIGH: In a real implementation, consider something far more robust than
	// this string comparison.
	return itemURL.Host == db.hostname, nil
}
