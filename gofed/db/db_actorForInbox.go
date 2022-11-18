package db

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/gofed/common"
	"github.com/benpate/derp"
)

func (db *Database) ActorForInbox(ctx context.Context, outboxURL *url.URL) (actorURL *url.URL, err error) {

	const location = "gofed.db.ActorForInbox"

	userID, _, _, err := common.ParseURL(outboxURL)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing inbox URL", outboxURL)
	}

	actorURL, err = url.Parse(common.ActorURL(db.hostname, userID))

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing actor URL", actorURL)
	}

	return actorURL, nil
}
