package service

import (
	"sort"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/sherlock"
)

func (service *Following) RefreshAndConnect(following model.Following) {

	// Try to refresh the Actor in the cache
	// nolint:errcheck
	service.activityStreams.Load(following.URL, sherlock.AsActor(), ascache.WithForceReload())

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
	actor, err := service.activityStreams.Load(following.URL, sherlock.AsActor())

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

	// Import the actor's outbox and messages
	outbox, err := actor.Outbox().Load()

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading outbox", actor))
		return
	}

	// Create a channel from this outbox...
	done := make(chan struct{})
	documentChan := collections.Documents(outbox, done)  // start reading documents from the outbox
	documentChan = channel.Limit(12, documentChan, done) // Limit to last 12 documents
	documents := channel.Slice(documentChan)             // Convert the channel into a slice

	// Sort the collection chronologically so that they're imported in the correct order.
	sort.Slice(documents, func(a int, b int) bool {
		return documents[a].Published().Before(documents[b].Published())
	})

	// Try to add each message into the database unitl done
	for _, document := range documents {

		// Try to save the document to the database.
		if err := service.SaveMessage(following, document); err != nil {
			derp.Report(derp.Wrap(err, location, "Error saving document", document))
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

	// Push services will not work via localhost :(
	if domain.IsLocalhost(service.host) {
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

// getActualDocument traverses "Create" and "Update" messages to get the actual document that we want to save
func getActualDocument(document streams.Document) streams.Document {

	// Load the full version of the document (if it's a link)
	loaded, err := document.Load()

	if err != nil {
		return document
	}

	switch loaded.Type() {

	// If the document is a "Create" activity, then we want to use the object as the actual message
	case vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate:
		return loaded.Object()

	// Otherwise, we'll just use the document as-is
	default:
		return loaded
	}
}
