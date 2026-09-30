package consumer

import (
	"errors"
	"net/http"
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// PollFollowing_Record polls an individual Following record for new post from its outbox (or RSS feed)
func PollFollowing_Record(factory *service.Factory, session data.Session, user *model.User, following *model.Following, args mapof.Any) queue.Result {

	const location = "consumer.PollFollowing_Record"

	// Load the Actor that we're following
	client := factory.ActivityStream().UserClient(user.UserID)
	actor, err := client.Load(following.URL)

	if err != nil {
		return actorError(factory, session, following, err)
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
		result, err := document.Load()

		if err != nil {

			// RULE: A 429 rate-limits the HOST, not this document, so every document left in this
			// window would hit it too.  requeue() reschedules the whole task after Retry-After.
			if isTooMany, _ := derp.IsTooManyRequests(err); isTooMany {
				return requeue(derp.Wrap(err, location, "Loading document", "following: "+following.URL))
			}

			// RULE: Any other 4xx is permanent for THIS document.  A malformed, relative, or missing
			// document ID fails identically on every future poll, so reporting re-files it each cycle.
			if derp.IsClientError(err) {
				log.Debug().Str("location", location).Str("following", following.URL).Str("document", document.ID()).Msg("Skipping unreadable document")
				continue
			}

			// Anything else may succeed on a later poll, and stays visible
			derp.Report(derp.Wrap(err, location, "Loading document", "following: "+following.URL, document.Value()))
			continue
		}

		// Save message to the Inbox
		if err := factory.Following().SaveNewsItem(session, following, result, model.OriginTypePrimary); err != nil {
			return queue.Error(derp.Wrap(err, location, "Saving NewsItem to NewsFeed", result.Value()))
		}
	}

	// Recalculate Folder unread counts
	if err := factory.Folder().CalculateUnreadCount(session, following.UserID, following.FolderID.Value()); err != nil {
		return queue.Error(derp.Wrap(err, location, "Recalculating unread count"))
	}

	// Stamp the poll and schedule the next one.  Without this, NextPoll never moves and
	// PollFollowing-Index re-selects the record on every sweep, whatever its PollDuration says.
	if err := factory.Following().SetStatusPollSuccess(session, following); err != nil {
		return queue.Error(derp.Wrap(err, location, "Recording successful poll"))
	}

	// Success!
	return queue.Success()
}

// actorError records a failed Actor load on the Following itself, and maps it onto a queue.Result
func actorError(factory *service.Factory, session data.Session, following *model.Following, err error) queue.Result {

	const location = "consumer.actorError"

	log.Debug().Str("location", location).Str("following", following.URL).Int("code", derp.ErrorCode(err)).Msg("Poll failed")

	// RULE: A 429 rate-limits the whole HOST, so it reschedules the task and leaves the Following
	// untouched.  Recording it would count the host's throttle as this record's failure.
	if isTooMany, _ := derp.IsTooManyRequests(err); isTooMany {
		return requeue(derp.Wrap(err, location, "Loading ActivityPub Actor", "following: "+following.URL))
	}

	// Record the failure, which the service turns into GONE, FAILURE, or (eventually) PAUSED
	if inner := factory.Following().SetStatusPollError(session, following, err); inner != nil {
		return queue.Error(derp.Wrap(inner, location, "Recording failed poll", "following: "+following.URL))
	}

	// Report only what nothing here can account for -- once per poll, not once per retry (BUG-148)
	if shouldReportPollError(err) {
		derp.Report(derp.Wrap(err, location, "Loading ActivityPub Actor", "following: "+following.URL))
	}

	return queue.Success()
}

// shouldReportPollError returns TRUE for a failed Actor load that nothing here can account for,
// and which therefore deserves a human's attention in the error log.
func shouldReportPollError(err error) bool {

	// RULE: A 4xx is permanent and understood, so it is not reported.
	if derp.IsClientError(err) {
		return false
	}

	// RULE: A 2xx that carried no Actor is understood too -- the server answered, and the
	// Following record now says so.  Reporting re-files that same fact on every poll (BUG-151).
	if answeredWithoutActor(err) {
		return false
	}

	// Anything else is unexpected, and stays visible
	return true
}

// answeredWithoutActor returns TRUE if a failed Actor load was a 2xx response that arrived intact
// but held no Actor.
func answeredWithoutActor(err error) bool {

	// The response that arrived is carried by the HTTPError, whatever code the wrapping added
	var httpError derp.HTTPError

	if !errors.As(err, &httpError) {
		return false
	}

	// RULE: Only a 2xx means the request completed.  A 4xx is a refusal, a 5xx is the server's
	// own fault, and a transport failure never produced a response to read at all.
	status := httpError.Response.StatusCode

	return (status >= http.StatusOK) && (status < http.StatusMultipleChoices)
}
