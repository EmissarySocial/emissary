package gofed

import (
	"context"

	"github.com/go-fed/activity/streams/vocab"
)

// Create stores the arbitrary ActivityStreams asType object into the database. It
// should be uniquely new to the database when examining its id property, and shouldn't
// overwrite any existing data.
//
// If needed, use streams.Serialize to turn the vocab.Type into literal JSON-LD bytes.
func (db Database) Create(c context.Context, asType vocab.Type) error {
	return nil
}

/*


https://emissary.social/@benpate/inbox/1234567890 => InboxService
https://emissary.social/@benpate/outbox/1234567890 => OutboxService













*/
