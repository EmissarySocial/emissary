package service

import (
	"sort"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/sherlock"
	"github.com/rs/zerolog/log"
)

func (service *Following) RefreshAndConnect(following model.Following) {

	// Try to refresh the Actor in the cache
	// nolint:errcheck
	service.activityService.Load(following.URL, sherlock.AsActor(), ascache.WithForceReload())

	// Try to connect the Following record
	if err := service.Connect(following); err != nil {
		derp.Report(derp.Wrap(err, "service.Following.RefreshAndConnect", "Error connecting to actor"))
		return
	}
}

// Connect attempts to connect to a new URL and determines how to follow it.
func (service *Following) Connect(following model.Following) error {

	const location = "service.Following.Connect"

	// Update the following status
	if err := service.SetStatusLoading(&following); err != nil {
		return derp.Wrap(err, location, "Error updating following status", following)
	}

	// Try to load the actor from the remote server.  Errors mean that this actor cannot
	// be resolved, so we should mark the Following as a "Failure".
	actor, err := service.activityService.Load(following.URL, sherlock.AsActor())

	if err != nil {
		if innerError := service.SetStatusFailure(&following, err.Error()); err != nil {
			return derp.Wrap(innerError, location, "Error updating following status", following)
		}
		return err
	}

	// Set values in the Following record...
	following.Label = actor.Name()
	following.ProfileURL = actor.ID()
	following.ImageURL = actor.IconOrImage().URL()

	// ...and mark the status as "Success"
	if err := service.SetStatusSuccess(&following); err != nil {
		return derp.Wrap(err, location, "Error setting status", following)
	}

	// Try to load an initial list of messages from the actor's outbox
	service.connect_LoadMessages(&following, &actor)

	// Try to connect to push services (WebSub, ActivityPub, etc)
	service.connect_PushServices(&following, &actor)

	// Kool-Aid man says "ooooohhh yeah!"
	return nil
}

func (service *Following) connect_LoadMessages(following *model.Following, actor *streams.Document) {

	const location = "service.Following.connect_LoadMessages"

	// Create a channel from this outbox...
	done := make(chan struct{})
	outbox := actor.Outbox()
	documentChan := collections.Documents(outbox, done)  // start reading documents from the outbox
	documentChan = channel.Limit(12, documentChan, done) // Limit to last 12 documents
	documents := channel.Slice(documentChan)             // Convert the channel into a slice

	// Sort the collection chronologically so that they're imported in the correct order.
	sort.Slice(documents, func(a int, b int) bool {
		return documents[a].Published().Before(documents[b].Published())
	})

	// Try to add each message into the database unitl done
	for _, document := range documents {

		// RULE: For RSS feeds, push this document down the stack once more, to:
		// 1. Retrieve any extra data missing from an RSS feed
		// 2. Guarantee that the document has been saved in our cache.
		// nolint:errcheck -- It's okay to ignore errors because pages may exist
		// in an RSS feed, but return an error to us right now. (e.g. CAPTCHAs)
		result, _ := document.Load(sherlock.WithDefaultValue(document.Map()))

		// Try to save the document to the database.
		if err := service.SaveMessage(following, result, model.OriginTypePrimary); err != nil {
			derp.Report(derp.Wrap(err, location, "Error saving document to Inbox", result.Value()))
		}
	}

	// Recalculate Folder unread counts
	if err := service.folderService.ReCalculateUnreadCountFromFolder(following.UserID, following.FolderID); err != nil {
		derp.Report(derp.Wrap(err, location, "Error recalculating unread count"))
	}
}

// connect_PushServices tries to connect to the best available push service
func (service *Following) connect_PushServices(following *model.Following, actor *streams.Document) {

	const location = "service.Following.connect_PushServices"

	log.Debug().Str("loc", location).Msg("Trying to connect to push services")

	// Prevent attempts to connect to external domains from localhost. It won't work anyway.
	if domain.IsLocalhost(service.host) && !domain.IsLocalhost(following.ProfileURL) {
		log.Debug().Str("loc", location).Msg("Cannot connect to external push services from localhost")
		return
	}

	// If this actor has an ActivityPub inbox, then try to via ActivityPub
	if inbox := actor.Inbox(); inbox.NotNil() {
		if ok, err := service.connect_ActivityPub(following, actor); ok {
			return
		} else {
			derp.Report(derp.Wrap(err, location, "Error connecting to ActivityPub"))
		}
	}

	// If a WebSub hub is defined, then use that.
	if hub := actor.Endpoints().Get("websub").String(); hub != "" {
		// TODO: LOW: Implement Fat Pings
		if ok, err := service.connect_WebSub(following, hub); ok {
			return
		} else {
			derp.Report(derp.Wrap(err, location, "Error connecting to WebSub"))
		}
	}

	// RSSCloud is TBD because WebSub seems to have won the war.
	// TODO: LOW: RSSCloud
}
