package service

import (
	"bytes"
	"mime"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/remote"
)

// Connect attempts to connect to a new URL and determines how to follow it.
func (service *Following) Connect(following model.Following) error {

	const location = "service.Following.Connect"

	// Update the following status
	if err := service.SetStatus(&following, model.FollowingStatusLoading, ""); err != nil {
		return derp.Wrap(err, location, "Error updating following status", following)
	}

	// Try to load the targetURL.
	var body bytes.Buffer
	transaction := remote.
		Get(following.URL).
		Header("Accept", followingMimeStack).
		Response(&body, nil)

	if err := transaction.Send(); err != nil {
		return derp.Wrap(err, location, "Error loading URL", following.URL)
	}

	mimeType := transaction.ResponseObject.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(mimeType)

	switch mediaType {

	// Handle JSONFeeds directly
	case model.MimeTypeJSONFeed:
		return service.import_JSONFeed(&following, transaction.ResponseObject, &body)

	// Handle Atom and RSS feeds directly
	case model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText:
		return service.import_RSS(&following, transaction.ResponseObject, &body)

	// Parse HTML to find feed links (and look for h-feed microformats)
	case model.MimeTypeHTML:
		return service.import_HTML(&following, transaction.ResponseObject, &body)
	}

	// Otherwise, we can't find a valid feed, so report an error.
	if err := service.SetStatus(&following, model.FollowingStatusFailure, "Unsupported content type: "+mimeType); err != nil {
		return derp.Wrap(err, location, "Error updating following status", following)
	}

	// Try to connect to push services (WebSub, ActivityPub, etc)
	service.connect_PushServices(&following)

	return derp.New(derp.CodeInternalError, location, "Unsupported content type", mimeType)
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
		return derp.Wrap(err, "service.Following", "Error updating following status", following)
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

// saveActivity adds/updates an individual Activity based on an RSS item
func (service *Following) saveActivity(following *model.Following, activity *model.Activity) error {

	const location = "service.Following.saveActivity"

	original := model.NewActivity()
	activity.UpdateWithFollowing(following)

	// Search for an existing Activity that matches the parameter
	err := service.inboxService.LoadByDocumentURL(following.UserID, activity.Document.URL, &original)

	switch {

	// If this activity IS NOT FOUND in the database, then save the new record to the database
	case derp.NotFound(err):

		if err := service.inboxService.Save(activity, "Activity Imported"); err != nil {
			return derp.Wrap(err, location, "Error saving activity")
		}

		return nil

	// If this activity IS FOUND in the database, then try to update it
	case err == nil:

		// Otherwise, update the original and save
		original.UpdateWithActivity(activity)

		if err := service.inboxService.Save(&original, "Activity Updated"); err != nil {
			return derp.Wrap(err, location, "Error saving activity")
		}

		return nil
	}

	// Otherwise, it's a legitimate error, so let's shut this whole thing down.
	return derp.Wrap(err, location, "Error loading local activity")
}

// connect_PushServices tries to connect to the best available push service
func (service *Following) connect_PushServices(following *model.Following) {

	// ActivityPub is first because it's the highest fidelity (when it works)
	if activityPub := following.GetLink("type", model.MimeTypeActivityPub); !activityPub.IsEmpty() {
		if err := service.connect_ActivityPub(following, activityPub); err == nil {
			return
		}
	}

	// WebSub is second because it works (and fat pings will be cool when they're implemented)
	if hub := following.GetLink("rel", model.LinkRelationHub); !hub.IsEmpty() {
		if self := following.GetLink("rel", model.LinkRelationSelf); !self.IsEmpty() {
			if err := service.connect_WebSub(following, hub, self.Href); err == nil {
				return
			}
		}
	}
}
