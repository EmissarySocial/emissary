package service

import (
	"context"
	"sort"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/sherlock"
	"github.com/rs/zerolog/log"
)

func (service *Following) RefreshAndConnect(session data.Session, following model.Following) error {

	const location = "service.Following.RefreshAndConnect"

	// Try to refresh the Actor in the cache
	activityService := service.factory.ActivityStream(model.ActorTypeUser, following.UserID)
	actor, err := activityService.Client().Load(following.URL, sherlock.AsActor(), ascache.WithForceReload())

	if err != nil {
		if err := service.markFailure(following); err != nil {
			return derp.Wrap(err, location, "Unable to refresh ActivityPub Actor; Unable to mark `Following` record as `Failure`")
		}
		return derp.Wrap(err, location, "Unable to refresh ActivityPub Actor")
	}

	// Try to connect the Following record
	if err := service.connect(session, actor, following); err != nil {
		return derp.Wrap(err, location, "Unable to connect to ActivityPub Actor")
	}

	// Success
	return nil
}

// Connect attempts to connect to a new URL and determines how to follow it.
func (service *Following) Connect(session data.Session, following model.Following) error {

	const location = "service.Following.Connect"

	// Try to load the Actor in the cache (allow cached values)
	activityService := service.factory.ActivityStream(model.ActorTypeUser, following.UserID)
	actor, err := activityService.Client().Load(following.URL, sherlock.AsActor(), ascache.WithForceReload())

	if err != nil {
		return derp.Wrap(err, location, "Unable to load ActivityPub Actor")
	}

	// Try to connect the Following record
	if err := service.connect(session, actor, following); err != nil {
		return derp.Wrap(err, location, "Unable to connect to ActivityPub Actor")
	}

	return nil
}

// Connect attempts to connect to a new URL and determines how to follow it.
func (service *Following) connect(session data.Session, actor streams.Document, following model.Following) error {

	const location = "service.Following.connect"

	// Set values in the Following record...
	following.Label = actor.Name()
	following.ProfileURL = actor.ID()
	following.IconURL = actor.IconOrImage().URL()
	following.Username = actor.UsernameOrID()

	// Update the following status
	if err := service.SetStatusLoading(session, &following); err != nil {
		return derp.Wrap(err, location, "Unable to set `Following` status to `Loading`", following)
	}

	//
	// TODO: Should the following async functions be moved to the task queue?
	//

	// Try to connect to push services (WebSub, ActivityPub, etc)
	go service.connect_PushServices(&following, &actor)

	// Try to load an initial list of messages from the actor's outbox
	go service.connect_LoadMessages(&following, &actor)

	// Kool-Aid man says "ooooohhh yeah!"
	return nil
}

// markFailure marks a `Following` record as failed when we are unable to connect to the
// ActivityPub actor
func (service *Following) markFailure(following model.Following) error {

	const location = "service.Following.markFailure"

	ctx, cancel := timeoutContext(1)
	defer cancel()

	_, err := service.factory.Server().WithTransaction(ctx, func(session data.Session) (any, error) { // nolint:scopeguard

		// Update the following status
		if err := service.SetStatusFailure(session, &following, "Unable to connect to ActivityPub Actor"); err != nil {
			return nil, derp.Wrap(err, location, "Unable to set `Following` status to `Failure`", following)
		}

		return nil, nil
	})

	if err != nil {
		return derp.Wrap(err, location, "Unable to mark `Following` record as `Failure`")
	}

	return nil
}

func (service *Following) connect_LoadMessages(following *model.Following, actor *streams.Document) {

	const location = "service.Following.connect_LoadMessages"

	// Create a channel from this outbox...
	outbox := actor.Outbox()
	documentRangeFunc := collections.RangeDocuments(outbox) // start reading documents from the outbox
	documentRangeFunc = ranges.Limit(12, documentRangeFunc) // Limit to last 12 documents
	documents := ranges.Slice(documentRangeFunc)            // Convert the channel into a slice

	// Sort the collection chronologically so that they're imported in the correct order.
	sort.Slice(documents, func(a int, b int) bool {
		return documents[a].Published().Before(documents[b].Published())
	})

	// Try to add each message into the database until done
	for _, document := range documents {

		// RULE: For RSS feeds, push this document down the stack once more, to:
		// 1. Retrieve any extra data missing from an RSS feed
		// 2. Guarantee that the document has been saved in our cache.
		// It's okay to ignore errors because pages may exist
		// in an RSS feed, but return an error to us right now. (e.g. CAPTCHAs)
		// nolint:errcheck
		result, _ := document.Load(sherlock.WithDefaultValue(document.Map()))

		// Unique transaction to save the document to the database.
		_, err := service.factory.Server().WithTransaction(context.Background(), func(session data.Session) (any, error) { // nolint:scopeguard (readability)

			if err := service.SaveMessage(session, following, result, model.OriginTypePrimary); err != nil {
				return nil, derp.Wrap(err, location, "Unable to save `Message` to Inbox", result.Value())
			}
			return nil, nil
		})

		if err != nil {
			derp.Report(err)
		}
	}

	// Recalculate Folder unread counts
	_, err := service.factory.Server().WithTransaction(context.Background(), func(session data.Session) (any, error) { // nolint:scopeguard
		if err := service.folderService.CalculateUnreadCount(session, following.UserID, following.FolderID); err != nil {
			return nil, derp.Wrap(err, location, "Unable to recalculate unread count")
		}
		return nil, nil
	})

	if err != nil {
		derp.Report(err)
	}
}

// connect_PushServices tries to connect to the best available push service
func (service *Following) connect_PushServices(following *model.Following, actor *streams.Document) {

	const location = "service.Following.connect_PushServices"

	log.Debug().Str("loc", location).Msg("Trying to connect to push services")

	// Prevent attempts to connect to external domains from localhost. It won't work anyway.
	if dt.IsLocalhost(service.host) && !dt.IsLocalhost(following.ProfileURL) {
		log.Debug().Str("loc", location).Msg("Unable to connect to external push services from localhost")
		return
	}

	// Create a new database transaction (for those about to rock)
	_, err := service.factory.Server().WithTransaction(context.Background(), func(session data.Session) (any, error) {

		// If this actor has an ActivityPub inbox, then try to via ActivityPub
		if inbox := actor.Inbox(); inbox.NotNil() {
			if ok, err := service.connect_ActivityPub(session, following, actor); ok {
				return nil, nil
			} else {
				return nil, derp.Wrap(err, location, "Unable to connect to ActivityPub")
			}
		}

		// If a WebSub hub is defined, then use that.
		if hub := actor.Endpoints().Get("websub").String(); hub != "" {
			if ok, err := service.connect_WebSub(following, hub); ok {
				return nil, nil
			} else {
				return nil, derp.Wrap(err, location, "Unable to connect to WebSub")
			}
		}

		return nil, nil
	})

	derp.Report(err)
}
