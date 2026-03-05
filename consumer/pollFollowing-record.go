package consumer

import (
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/sherlock"
	"github.com/benpate/turbine/queue"
)

// PollFollowing_Record polls an individual Following record for new post from its outbox (or RSS feed)
func PollFollowing_Record(factory *service.Factory, session data.Session, user *model.User, following *model.Following, args mapof.Any) queue.Result {

	const location = "consumer.PollFollowing_Record"

	// Load the Actor that we're following
	client := factory.ActivityStream().UserClient(user.UserID)
	actor, err := client.Load(following.URL, sherlock.AsActor())

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to load ActivityPub Actor", "url: "+following.URL))
	}

	// Create a channel from this outbox...
	outbox := actor.Outbox()
	documentRangeFunc := collections.RangeDocuments(outbox) // start reading activities from the outbox
	documentRangeFunc = ranges.Limit(24, documentRangeFunc) // Limit to last 24 activities
	documentSlice := ranges.Slice(documentRangeFunc)        // Convert the iterator into a slice
	documents := slices.Backward(documentSlice)             // Read documents from the slice (oldest to newest)

	// Try to add each message into the database until done
	for _, document := range documents {

		// Try to load the document from the Actor's outbox
		result, err := document.Load(sherlock.WithDefaultValue(document.Map()))

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to load document", document.Value()))
			continue
		}

		// Save message to the Inbox
		if err := factory.Following().SaveMessage(session, following, result, model.OriginTypePrimary); err != nil {
			return queue.Error(derp.Wrap(err, location, "Unable to save Message to Inbox", result.Value()))
		}
	}

	// Recalculate Folder unread counts
	if err := factory.Folder().CalculateUnreadCount(session, following.UserID, following.FolderID.Value()); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to recalculate unread count"))
	}

	// Success!
	return queue.Success()
}
