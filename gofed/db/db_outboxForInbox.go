package db

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/gofed/common"
	"github.com/benpate/derp"
)

func (db *Database) OutboxForInbox(ctx context.Context, inboxURL *url.URL) (outboxURL *url.URL, err error) {

	const location = "gofed.db.OutboxForInbox"

	userID, _, _, err := common.ParseURL(inboxURL)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing inbox URL", inboxURL)
	}

	outboxURL, err = url.Parse(common.ActorOutboxURL(db.hostname, userID))

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing outbox URL", outboxURL)
	}

	return outboxURL, nil
}
