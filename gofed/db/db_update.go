package db

import (
	"context"

	"github.com/go-fed/activity/streams/vocab"
)

func (db *Database) Update(_ context.Context, item vocab.Type) error {
	return db.save(item, "Update via ActivityPub")
}
