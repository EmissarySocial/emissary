package consumer

import (
	"net/http"
	"slices"
	"strconv"
	"unicode/utf8"

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

// pollOutcome names what a failed poll means for the Following record it happened to
type pollOutcome int

const (
	// pollOutcomeRateLimited means the HOST is throttling us, and this record did nothing wrong
	pollOutcomeRateLimited pollOutcome = iota

	// pollOutcomeGone means the source is gone for good, on the remote server's own say-so
	pollOutcomeGone

	// pollOutcomeFailed means "record it and try again at the normal cadence"
	pollOutcomeFailed
)

// classifyPollError decides what a failed Actor load means for the Following record
func classifyPollError(err error) pollOutcome {

	// RULE: The 429 test MUST come first.  derp.IsClientError covers 400-499, so it covers 429
	// too, and testing it first would turn every rate limit into the record's own failure.
	if isTooMany, _ := derp.IsTooManyRequests(err); isTooMany {
		return pollOutcomeRateLimited
	}

	// RULE: A 410 is the remote server stating the account is DELETED, not a guess we are
	// making.  Mastodon answers 410 for a deleted account, so this needs no waiting period.
	if derp.ErrorCode(err) == http.StatusGone {
		return pollOutcomeGone
	}

	// RULE: Everything else -- 4xx, 5xx, DNS, TLS, timeout -- is RECORDED rather than retried.
	// A dead domain never answers 4xx, so classifying only client errors left exactly the
	// case that matters touching nothing at all.  Recording is what eventually reaches PAUSED.
	return pollOutcomeFailed
}

// actorError records a failed Actor load on the Following itself, and maps it onto a queue.Result
func actorError(factory *service.Factory, session data.Session, following *model.Following, err error) queue.Result {

	const location = "consumer.actorError"

	outcome := classifyPollError(err)
	log.Debug().Str("location", location).Str("following", following.URL).Int("code", derp.ErrorCode(err)).Int("outcome", int(outcome)).Msg("Poll failed")

	switch outcome {

	case pollOutcomeRateLimited:
		return requeue(derp.Wrap(err, location, "Loading ActivityPub Actor", "following: "+following.URL))

	case pollOutcomeGone:
		if inner := factory.Following().SetStatusGone(session, following, followingStatusMessage(err)); inner != nil {
			return queue.Error(derp.Wrap(inner, location, "Marking Following as gone", "following: "+following.URL))
		}

		return queue.Success()
	}

	// Record the failure, which is also what eventually escalates the record to PAUSED
	if inner := factory.Following().SetStatusPollFailure(session, following, followingStatusMessage(err)); inner != nil {
		return queue.Error(derp.Wrap(inner, location, "Marking Following as failed", "following: "+following.URL))
	}

	// RULE: A 4xx is permanent and understood, so it is not reported.  Anything else is
	// unexpected and stays visible -- once per poll now, not once per retry (BUG-148).
	if !derp.IsClientError(err) {
		derp.Report(derp.Wrap(err, location, "Loading ActivityPub Actor", "following: "+following.URL))
	}

	return queue.Success()
}

// statusMessageMaxLength matches the "statusMessage" schema in model.Following, which rejects
// anything longer.  A root message quoted from a transport failure has no length bound of its own.
const statusMessageMaxLength = 1024

// followingStatusMessage renders a failed poll as a short sentence for the Following's owner
func followingStatusMessage(err error) string {

	code := derp.ErrorCode(err)

	switch code {

	case http.StatusUnauthorized, http.StatusForbidden:
		return "This account refused our request (" + strconv.Itoa(code) + "). It may be private, or may have blocked this server."

	case http.StatusNotFound:
		return "This account could not be found (" + strconv.Itoa(code) + "). It may have been moved or deleted."

	case http.StatusGone:
		return "This account has been deleted (410)."
	}

	if derp.IsClientError(err) {
		return "Unable to read this account (" + strconv.Itoa(code) + ")."
	}

	// RULE: Everything else may be a real 5xx or a transport failure that never reached a
	// server, and derp reports both as 500 -- so quote the reason instead of naming a code.
	return truncate("Could not reach this server: "+derp.RootMessage(err), statusMessageMaxLength)
}

// truncate shortens a string to at most `maxLength` bytes, marking any text it removed
func truncate(value string, maxLength int) string {

	if len(value) <= maxLength {
		return value
	}

	// RULE: Below the width of the marker there is no room to mark anything, so cut hard
	suffix := "..."

	if maxLength < len(suffix) {
		suffix = ""
	}

	cut := maxLength - len(suffix)

	// A negative length would panic on the slice below
	if cut < 0 {
		cut = 0
	}

	// Step back to a rune boundary so the result is never invalid UTF-8
	for (cut > 0) && !utf8.RuneStart(value[cut]) {
		cut--
	}

	return value[:cut] + suffix
}
