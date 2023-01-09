package gofed

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// InboxContains return s TRUE if the ActivityStreams object with the id is contained
// within that Inbox's OrderedCollection.
//
// A naive implementation may just do a linear search through the OrderedCollection,
// while certain databases may permit better lookup performance with a proper query.
func (db Database) InboxContains(c context.Context, inbox *url.URL, id *url.URL) (contains bool, err error) {

	const location = "gofed.Database.InboxContains"

	// Parse the URLs provided
	inboxOwnerID, inboxLocation, _, err := ParseURL(inbox)

	if err != nil {
		return false, derp.Wrap(err, location, "Error parsing Inbox URL", inbox)
	}

	activityOwnerID, activityLocation, activityID, err := ParseURL(id)

	if err != nil {
		return false, derp.Wrap(err, location, "Error parsing Activity URL", id)
	}

	// Validate basic assumptions about the URLs
	if inboxOwnerID != activityOwnerID {
		return false, nil
	}

	if inboxLocation != model.ActivityLocationInbox {
		return false, derp.NewInternalError("InboxContains", "Inbox URL is not an Inbox", inbox.String())
	}

	if activityLocation != model.ActivityLocationInbox {
		return false, derp.NewInternalError("InboxContains", "Activity URL is not an Inbox", inbox.String())
	}

	// Try to load the Activity from the database
	activity := model.NewActivity()
	err = db.activityService.LoadInboxActivity(inboxOwnerID, activityID, &activity)

	// If NO error, then EXISTS
	if err == nil {
		return true, nil
	}

	// If NOT FOUND error, then FALSE
	if derp.NotFound(err) {
		return false, nil
	}

	// Otherwise, REAL Error, FAIL
	return false, derp.Wrap(err, "gofed.Database.InboxContains", "Error loading activity", activityID)
}
