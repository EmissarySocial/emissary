package db

import (
	"context"
	"errors"
	"net/url"
	"sync"
)

func (db *Database) Unlock(ctx context.Context, id *url.URL) error {
	// Once Go-Fed is done calling Database methods, the relevant `id`
	// entries are unlocked.

	i, ok := db.locks.Load(id.String())
	if !ok {
		return errors.New("Missing an id in Unlock")
	}
	mu := i.(*sync.Mutex)
	mu.Unlock()
	return nil
}
