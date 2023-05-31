package service

import (
	"bytes"
	"mime"
	"net/mail"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/remote"
)

// Connect attempts to connect to a new URL and determines how to follow it.
func (service *Following) Connect(following model.Following) error {

	const location = "service.Following.Connect"

	// Update the following status
	if err := service.SetStatus(&following, model.FollowingStatusLoading, ""); err != nil {
		return derp.Wrap(err, location, "Error updating following status", following)
	}

	// If there is an error connecting to the URL, then mark the status as Failure
	if err := service.connect(&following); err != nil {

		if innerError := service.SetStatus(&following, model.FollowingStatusFailure, err.Error()); err != nil {
			return derp.Wrap(innerError, location, "Error updating following status", following)
		} else {
			return err
		}
	}

	// Otherwise, mark the status as "Connected"
	if err := service.SetStatus(&following, model.FollowingStatusSuccess, ""); err != nil {
		return derp.Wrap(err, location, "Error setting status", following)
	}

	// Finally, look for push services to connect to (WebSub, ActivityPub, etc)
	service.connect_PushServices(&following)

	return nil
}

func (service *Following) connect(following *model.Following) error {

	const location = "service.Following.connect"

	// Try to use the following.URL as an email address (or @username@server.name)
	emailAddress := strings.TrimPrefix(following.URL, "@")

	if email, err := mail.ParseAddress(emailAddress); err == nil {

		resource, err := digit.Lookup(email.Address)

		if err != nil {
			return derp.Wrap(err, location, "Error looking up email address", email.Address)
		}

		following.Links = resource.Links
		return nil
	}

	// Try to use the following.URL as an actual URL.

	// Add protocol to the URL if it's missing
	following.URL = domain.AddProtocol(following.URL)

	// Try to connect to the remote server
	var body bytes.Buffer
	transaction := remote.
		Get(following.URL).
		Header("Accept", followingMimeStack).
		Response(&body, nil)

	if err := transaction.Send(); err != nil {
		return derp.Wrap(err, location, "Error connecting to remote website: "+following.URL)
	}

	// Look for Links to ActivityPub/Feeds/Hubs
	following.Links = discoverLinks(transaction.ResponseObject, &body)

	// Fall through means the remote server does not support ActivityPub.

	// Inspect the Content-Type header to determine how to parse the response.
	mimeType := transaction.ResponseObject.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(mimeType)

	switch mediaType {

	// NO-OP for ActivityPub here.  We will subscribe as a "push service"
	// at the end of the connect method
	case model.MimeTypeActivityPub:
		return nil

	// Handle JSONFeeds directly
	case model.MimeTypeJSONFeed, model.MimeTypeJSON:
		if err := service.import_JSONFeed(following, transaction.ResponseObject, &body); err != nil {
			return derp.Wrap(err, location, "Error importing JSONFeed", following.URL)
		}

	// Handle Atom and RSS feeds directly
	case model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText:
		if err := service.import_RSS(following, transaction.ResponseObject, &body); err != nil {
			return derp.Wrap(err, location, "Error importing RSS", following.URL)
		}

	// Parse HTML to find feed links (and look for h-feed microformats)
	case model.MimeTypeHTML:
		if err := service.import_HTML(following, transaction.ResponseObject, &body); err != nil {
			return derp.Wrap(err, location, "Error importing HTML", following.URL)
		}

	// Otherwise, we can't find a valid feed, so report an error.
	default:
		return derp.New(derp.CodeInternalError, location, "Unsupported content type: "+mimeType)
	}

	// Kool-Aid man says "ooooohhh yeah!"
	return nil
}

func (service *Following) Disconnect(following *model.Following) {

	switch following.Method {
	case model.FollowMethodActivityPub:
		service.disconnect_ActivityPub(following)

	case model.FollowMethodWebSub:
		service.disconnect_WebSub(following)
	}
}

// SetStatus updates the status (and statusMessage) of a Following record.
func (service *Following) SetStatus(following *model.Following, status string, statusMessage string) error {

	// RULE: Default Poll Duration is 24 hours
	if following.PollDuration == 0 {
		following.PollDuration = 24
	}

	// RULE: Require that poll duration is at least 1 hour
	if following.PollDuration < 1 {
		following.PollDuration = 1
	}

	// Update properties of the Following
	following.Status = status
	following.StatusMessage = statusMessage

	// Recalculate the next poll time
	switch following.Status {
	case model.FollowingStatusSuccess:

		// On success, "LastPolled" is only updated when we're successful.  Reset other times.
		following.LastPolled = time.Now().Unix()
		following.NextPoll = following.LastPolled + int64(following.PollDuration*60)
		following.ErrorCount = 0

	case model.FollowingStatusFailure:

		// On failure, compute exponential backoff
		// Wait times are 1m, 2m, 4m, 8m, 16m, 32m, 64m, 128m, 256m
		// But do not change "LastPolled" because that is the last time we were successful
		errorBackoff := following.ErrorCount

		if errorBackoff > 8 {
			errorBackoff = 8
		}

		errorBackoff = 2 ^ errorBackoff

		following.NextPoll = time.Now().Add(time.Duration(errorBackoff) * time.Minute).Unix()
		following.ErrorCount++

	default:
		// On all other statuse, the error counters are not touched
		// because "New" and "Loading" are going to be overwritten very soon.
	}

	// Try to save the Following to the database
	if err := service.collection.Save(following, "Updating status"); err != nil {
		return derp.Wrap(err, "service.Following.SetStatus", "Error updating following status", following)
	}

	// Success!!
	return nil
}

// poll loads the designated link, then uses the import function to import it into the database.
func (service *Following) poll(following *model.Following, link digit.Link, importFunc followingImportFunc) error {

	const location = "service.Following.poll"

	// Build the remote request.  Request the MediaType that was specified in the original link.
	var body bytes.Buffer

	transaction := remote.Get(link.Href).
		Header("Accept", link.MediaType).
		Response(&body, nil)

	if err := transaction.Send(); err != nil {
		return derp.Wrap(err, location, "Error fetching feed", link.Href)
	}

	if err := importFunc(following, transaction.ResponseObject, &body); err != nil {
		return derp.Wrap(err, location, "Error importing feed", link)
	}

	return nil
}

// saveToInbox adds/updates an individual Message based on an RSS item
func (service *Following) saveDocument(following *model.Following, document *streams.Document) error {

	const location = "service.Following.saveDocument"

	message := model.NewMessage()

	// Search for an existing Message that matches the parameter
	if err := service.inboxService.LoadByURL(following.UserID, document.ID(), &message); err != nil {
		if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error loading message")
		}
	}

	// Set/Update the message with information from the ActivityStream
	if message.IsNew() {
		message.UserID = following.UserID
		message.FolderID = following.FolderID
		message.Origin = following.Origin()
	}

	message.URL = document.ID()
	message.Label = document.Name()
	message.Summary = document.Summary()
	message.ImageURL = document.Image().URL()
	message.AttributedTo = convert.ActivityPubPersonLinks(document.AttributedTo())
	message.ContentHTML = document.Content()

	// Save the message to the database
	if err := service.inboxService.Save(&message, "Message Imported"); err != nil {
		return derp.Wrap(err, location, "Error saving message")
	}

	// Yee. Haw.
	return nil
}

// saveToInbox adds/updates an individual Message based on an RSS item
func (service *Following) saveToInbox(following *model.Following, message *model.Message) error {

	const location = "service.Following.saveToInbox"

	original := model.NewMessage()
	message.UpdateWithFollowing(following)

	// Search for an existing Message that matches the parameter
	err := service.inboxService.LoadByURL(following.UserID, message.URL, &original)

	switch {

	// If this message IS NOT FOUND in the database, then save the new record to the database
	case derp.NotFound(err):

		if err := service.inboxService.Save(message, "Message Imported"); err != nil {
			return derp.Wrap(err, location, "Error saving message")
		}

		return nil

	// If this message IS FOUND in the database, then try to update it
	case err == nil:

		// Otherwise, update the original and save
		original.UpdateWithMessage(message)

		if err := service.inboxService.Save(&original, "Message Updated"); err != nil {
			return derp.Wrap(err, location, "Error saving message")
		}

		return nil
	}

	// Otherwise, it's a legitimate error, so let's shut this whole thing down.
	return derp.Wrap(err, location, "Error loading local message")
}

// connect_PushServices tries to connect to the best available push service
func (service *Following) connect_PushServices(following *model.Following) {

	// ActivityPub is handled first because it is the highest fidelity connection
	if success, err := service.connect_ActivityPub(following); success {
		return
	} else if err != nil {
		derp.Report(err)
	}

	// WebSub is second because it works (and fat pings will be cool when they're implemented)
	// TODO: LOW: Implement Fat Pings
	if hub := following.GetLink("rel", model.LinkRelationHub); !hub.IsEmpty() {
		if err := service.connect_WebSub(following, hub); err != nil {
			derp.Report(err)
		} else {
			return
		}
	}

	// RSSCloud is TBD because WebSub seems to have won the war.
	// TODO: LOW: RSSCloud
}
