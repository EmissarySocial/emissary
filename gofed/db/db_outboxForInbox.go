package db

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/gofed/common"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

func (db *Database) OutboxForInbox(ctx context.Context, inboxURL *url.URL) (outboxURL *url.URL, err error) {

	const location = "gofed.db.OutboxForInbox"

	// Get the userID from the inbox URL
	userID, _, _, err := common.ParseURL(inboxURL)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing inbox URL", inboxURL)
	}

	// Load the user from the database
	user := model.NewUser()
	if err := db.userService.LoadByID(userID, &user); err != nil {
		return nil, derp.Wrap(err, location, "Error loading user", userID)
	}

	// Get the OutboxURL for the User
	outboxURL, err = url.Parse(user.ActivityPubOutboxURL(db.hostname))

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing outbox URL", outboxURL)
	}

	// Success!
	return outboxURL, nil
}
