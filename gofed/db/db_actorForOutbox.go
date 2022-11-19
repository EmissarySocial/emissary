package db

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/gofed/common"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

func (db *Database) ActorForOutbox(ctx context.Context, outboxURL *url.URL) (actorURL *url.URL, err error) {
	const location = "gofed.db.ActorForOutbox"

	// Get the userID from the Outbox URL
	userID, _, _, err := common.ParseURL(outboxURL)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing outbox URL", outboxURL)
	}

	// Load the user from the database
	user := model.NewUser()
	if err := db.userService.LoadByID(userID, &user); err != nil {
		return nil, derp.Wrap(err, location, "Error loading user", userID)
	}

	// Get the Profile URL for the User
	actorURL, err = url.Parse(user.ActivityPubProfileURL(db.hostname))

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing actor URL", actorURL)
	}

	// Success!
	return actorURL, nil
}
