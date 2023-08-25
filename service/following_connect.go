package service

import (
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/sherlock"
	"github.com/davecgh/go-spew/spew"
)

// Connect attempts to connect to a new URL and determines how to follow it.
func (service *Following) Connect(following model.Following) error {

	const location = "service.Following.Connect"

	isNewFollowing := (following.Status == model.FollowingStatusNew)
	// Update the following status
	if err := service.SetStatusLoading(&following); err != nil {
		return derp.Wrap(err, location, "Error updating following status", following)
	}

	// Try to load the actor from the remote server.  Errors mean that this actor cannot
	// be resolved, so we should mark the Following as a "Failure".
	actor, err := service.httpClient.LoadActor(following.URL)

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
	following.Format = strings.ToUpper(actor.Meta().GetString("format"))

	// ...and mark the status as "Success"
	if err := service.SetStatusSuccess(&following); err != nil {
		return derp.Wrap(err, location, "Error setting status", following)
	}

	// Try to connect to push services (WebSub, ActivityPub, etc)
	go service.connect_PushServices(&following, &actor)

	// Import the actor's outbox and messages
	outbox := actor.Outbox()
	done := make(chan struct{})
	documents := collections.Documents(outbox, done)
	counter := 0

	// Try to add each message into the database unitl done
	for documentOrLink := range documents {

		document := getActualDocument(documentOrLink)

		// RULE: For new following records, the first six records are "unread".  All others are "read"
		markRead := !isNewFollowing || (counter > 6)
		counter++

		// Try to save the document to the database.
		if err := service.saveMessage(following, document, markRead); err != nil {
			derp.Report(derp.Wrap(err, location, "Error saving document", document))
		}

		// Check business rules to see if we're done importing
		if service.cancelImport(following.Format, counter) {
			close(done)
			break
		}
	}

	// Recalculate Folder unread counts
	if err := service.folderService.ReCalculateUnreadCountFromFolder(following.UserID, following.FolderID); err != nil {
		return derp.Wrap(err, location, "Error recalculating unread count")
	}

	// Kool-Aid man says "ooooohhh yeah!"
	return nil
}

// cancelImport is a small test that runs every message imported by the Connect method.
// It returns TRUE if the import should be canceled, based on the following rules:
// 1. For ActivityStreams data, we import that last 256 messages from the outbox
// 2. For other records (RSS/Atom/JSONFeed/MicroFormats), we import all messages from the feed
func (service *Following) cancelImport(format string, counter int) bool {

	if format == model.FollowingFormatActivityStream {
		return (counter > 256)
	}

	return false
}

// saveToInbox adds/updates an individual Message based on an RSS item.  It returns TRUE if a new record was created
func (service *Following) saveMessage(following model.Following, document streams.Document, markRead bool) error {

	const location = "service.Following.saveMessage"
	message := model.NewMessage()

	// Search for an existing Message that matches the parameter
	if err := service.inboxService.LoadByURL(following.UserID, document.ID(), &message); err != nil {
		if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error loading message")
		}
	}

	// Load and refine the document from its actual URL
	document, err := service.httpClient.LoadDocument(document.ID(), document.Map())

	if err != nil {
		wrappedError := derp.Wrap(err, location, "Error loading document from source URL")
		if isDocumentAdequate(document) {
			return wrappedError
		} else {
			derp.Report(wrappedError)
		}
	}

	// Populate the new message
	message.UserID = following.UserID
	message.FolderID = following.FolderID
	message.Origin = following.Origin()
	message.URL = document.ID()
	message.Label = document.Name()
	message.Summary = document.Summary()
	message.ImageURL = document.Image().URL()
	message.AttributedTo = convert.ActivityPubPersonLinks(document.AttributedTo())
	message.ContentHTML = document.Content()
	message.PublishDate = document.Published().Unix()

	if markRead {
		message.ReadDate = message.PublishDate
	}

	// Save the message to the database
	if err := service.inboxService.Save(&message, "Message Imported"); err != nil {
		return derp.Wrap(err, location, "Error saving message")
	}

	spew.Dump("saveMessage: "+time.Unix(message.PublishDate, 0).Format(time.RFC3339), markRead, message.URL, message.Label, message.ReadDate)

	// Yee. Haw.
	return nil
}

// connect_PushServices tries to connect to the best available push service
func (service *Following) connect_PushServices(following *model.Following, actor *streams.Document) {

	// ActivityPub is handled first because it is the highest fidelity connection
	if actor.MetaString("format") == sherlock.FormatActivityStream {
		if ok, err := service.connect_ActivityPub(following, actor); ok {
			return
		} else if err != nil {
			derp.Report(derp.Wrap(err, "service.Following.connect_PushServices", "Error connecting to ActivityPub"))
		}
	}

	// WebSub is second because it works (and fat pings will be cool when they're implemented)
	// TODO: LOW: Implement Fat Pings
	if webSub := actor.MetaString("websub"); webSub != "" {
		if err := service.connect_WebSub(following, webSub); err != nil {
			derp.Report(derp.Wrap(err, "service.Following.connect_PushServices", "Error connecting to WebSub"))
		}
	}

	// RSSCloud is TBD because WebSub seems to have won the war.
	// TODO: LOW: RSSCloud
}

// getActualDocument traverses "Create" and "Update" messages to get the actual document that we want to save
func getActualDocument(document streams.Document) streams.Document {

	// Load the full version of the document (if it's a link)
	document = document.Document()

	switch document.Type() {

	// If the document is a "Create" activity, then we want to use the object as the actual message
	case vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate:
		return document.Object()

	// Otherwise, we'll just use the document as-is
	default:
		return document
	}
}

func isDocumentAdequate(document streams.Document) bool {

	if document.ID() == "" {
		return false
	}

	if document.Name() == "" {
		return false
	}

	if (document.Summary() == "") || (document.Content() == "") {
		return false
	}

	return true
}
