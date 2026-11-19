package service

import (
	"maps"
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

// TaskSendDeleteQuoteAuthorization is the name of the queue task that revokes a quote, by finding
// the quoting actor and sending a Delete of the QuoteAuthorization.
const TaskSendDeleteQuoteAuthorization = "SendDeleteQuoteAuthorization"

// Single-recipient outbound sends, queued post-commit.
//
// These replace the synchronous outbox.Actor.Send{Accept,Follow} calls that used to fire inside
// the request transaction. Each builds the same activity the outbox.Actor builder did, ADDS the
// recipient to the activity's `to` field (so hannibal/sender can resolve the inbox), and hands it
// to the queue via postcommit.Publish — so the signed HTTP delivery happens after commit and is
// independently retryable. The signing key is resolved later, in the sender consumer, via
// SendLocator.Actor(actorURL) — which is why F1 taught that resolver every actor type.
// See POST-COMMIT-FEDERATION.md F3.

// SendAccept queues an "Accept" activity addressed to the accepted activity's actor (typically the
// sender of a Follow request). actorURL is the accepting local actor's canonical URL (actor.ActorID()).
func (service *Outbox) SendAccept(session data.Session, actorURL string, acceptID string, activity streams.Document) {

	accept := mapof.Any{
		vocab.AtContext:      vocab.ContextTypeActivityStreams,
		vocab.PropertyID:     acceptID,
		vocab.PropertyType:   vocab.ActivityTypeAccept,
		vocab.PropertyActor:  actorURL,
		vocab.PropertyObject: activity.Map(),
		// The Accept is delivered to the actor(s) of the accepted activity — i.e. whoever sent the
		// Follow. outbox.Actor.SendAccept sent to activity.Actor().RangeIDs(); mirror that here.
		vocab.PropertyTo: slices.Collect(activity.Actor().RangeIDs()),
	}

	postcommit.Publish(session, service.queue, sender.OutboxSendToAllRecipients, accept)
}

// SendFollow queues a "Follow" activity addressed to remoteActorID (the actor being followed).
// actorURL is the following local actor's canonical URL (actor.ActorID()).
func (service *Outbox) SendFollow(session data.Session, actorURL string, followID string, remoteActorID string) {

	follow := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyID:        followID,
		vocab.PropertyType:      vocab.ActivityTypeFollow,
		vocab.PropertyActor:     actorURL,
		vocab.PropertyObject:    remoteActorID,
		vocab.PropertyPublished: datetime.Now(),
		// Address the Follow to the followed actor so the sender resolves their inbox. (The old
		// outbox.Actor.SendFollow passed the recipient out-of-band; the sender pipeline reads it
		// from the activity's `to`.)
		vocab.PropertyTo: []string{remoteActorID},
	}

	postcommit.Publish(session, service.queue, sender.OutboxSendToAllRecipients, follow)
}

// SendUndoFollow queues an "Undo" of a Follow activity, addressed to the followed remote actor.
// actorURL is the local follower's canonical URL (actor.ActorID()); follow is the original Follow
// activity (Following.AsJSONLD). The Follow is embedded as the Undo's object so the remote server
// knows which Follow to reverse.
//
// ADDRESSING NOTE (2026-07-14, review): this delivers the Undo to the remote actor being unfollowed
// (follow.object). The prior synchronous path (outbox.Actor.SendUndo) addressed the Undo to the
// Follow's RangeAddressees, which for our Follow JSON-LD is only the local actor itself — and
// outbox.Actor.Send skips self, so the outbound unfollow reached NOBODY. Delivering to the followed
// actor is the spec-correct behavior (https://www.w3.org/TR/activitypub/#undo-activity-outbox) and
// the entire point of an unfollow. This is a behavior FIX; flagged for review.
// See POST-COMMIT-FEDERATION.md F4.
func (service *Outbox) SendUndoFollow(session data.Session, actorURL string, follow mapof.Any) {

	undo := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyType:      vocab.ActivityTypeUndo,
		vocab.PropertyActor:     actorURL,
		vocab.PropertyObject:    follow,
		vocab.PropertyPublished: datetime.Now(),
		vocab.PropertyTo:        []string{follow.GetString(vocab.PropertyObject)},
	}

	postcommit.Publish(session, service.queue, sender.OutboxSendToAllRecipients, undo)
}

// SendAnnounce queues an "Announce" (boost) of object, addressed as a standard public boost:
// to:[Public], cc:[followersURL]. actorURL is the boosting actor's canonical URL (actor.ActorID());
// followersURL is that actor's followers-collection URI, which SendLocator.Recipient resolves to
// each ActivityPub follower's inbox (F1 signs as the actor; W6 taught the resolver stream/search
// followers). Replaces the synchronous outbox.Actor.SendAnnounce — delivery is now post-commit and
// per-recipient retryable. See POST-COMMIT-FEDERATION.md F3 (W6, option B).
//
// NOTE: the old outbox.Actor.SendAnnounce ALSO delivered to the boosted object's own addressees
// (object.RangeAddressees()); this standard-boost addressing intentionally does not — a boost
// reaches the booster's followers + Public, not the original post's private audience.
func (service *Outbox) SendAnnounce(session data.Session, actorURL string, announceID string, object streams.Document, followersURL string) {

	announce := mapof.Any{
		vocab.AtContext:         vocab.ContextTypeActivityStreams,
		vocab.PropertyID:        announceID,
		vocab.PropertyType:      vocab.ActivityTypeAnnounce,
		vocab.PropertyActor:     actorURL,
		vocab.PropertyObject:    object.Map(),
		vocab.PropertyPublished: datetime.Now(),
		vocab.PropertyTo:        []string{vocab.NamespacePublic},
		vocab.PropertyCC:        []string{followersURL},
	}

	postcommit.Publish(session, service.queue, sender.OutboxSendToAllRecipients, announce)
}

// SendAcceptQuote queues an "Accept" of a FEP-044f QuoteRequest, addressed to the quoting actor. Its
// result names the QuoteAuthorization that approves the quote. actorURL is the quoted author's URL.
func (service *Outbox) SendAcceptQuote(session data.Session, actorURL string, quoteRequest streams.Document, authorizationURL string) {
	accept := acceptQuoteActivity(actorURL, quoteRequest, authorizationURL)
	postcommit.Publish(session, service.queue, sender.OutboxSendToAllRecipients, accept)
}

// acceptQuoteActivity builds the "Accept" that SendAcceptQuote sends, in the shape of Mastodon's
// AcceptQuoteRequestSerializer.
func acceptQuoteActivity(actorURL string, quoteRequest streams.Document, authorizationURL string) mapof.Any {

	// The QuoteRequest is embedded by reference only: Mastodon matches an Accept to its request by
	// id, and re-publishing the inlined quote post under this actor's signature would vouch for it.
	return mapof.Any{
		vocab.AtContext:     sliceof.Any{vocab.ContextTypeActivityStreams, vocab.ContextTypeQuotePosts},
		vocab.PropertyID:    authorizationURL + "#accept",
		vocab.PropertyType:  vocab.ActivityTypeAccept,
		vocab.PropertyActor: actorURL,
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyID:         quoteRequest.ID(),
			vocab.PropertyType:       vocab.ActivityTypeQuoteRequest,
			vocab.PropertyActor:      quoteRequest.ActorID(),
			vocab.PropertyObject:     quoteRequest.Object().ID(),
			vocab.PropertyInstrument: quoteRequest.Instrument().ID(),
		},
		vocab.PropertyResult: authorizationURL,
		vocab.PropertyTo:     []string{quoteRequest.ActorID()},
	}
}

// SendDeleteQuoteAuthorization queues the task that revokes a quote by sending a "Delete" of its
// QuoteAuthorization.  The task looks up the quoting actor from the quote post, and retries while
// that post's server is unavailable, so the caller never waits on a remote server.
func (service *Outbox) SendDeleteQuoteAuthorization(session data.Session, actorURL string, authorization mapof.Any, followersURL string) {

	// The arguments are plain strings, which the stored queue returns unchanged
	postcommit.Publish(session, service.queue, TaskSendDeleteQuoteAuthorization, mapof.Any{
		"hostname":      service.host,
		"actor":         actorURL,
		"followers":     followersURL,
		"authorization": authorization.GetString(vocab.PropertyID),
		"attributedTo":  authorization.GetString(vocab.PropertyAttributedTo),
		"quote":         authorization.GetString(vocab.PropertyInteractingObject),
		"target":        authorization.GetString(vocab.PropertyInteractionTarget),
	})
}

// DeliverDeleteQuoteAuthorization queues delivery of the "Delete" described by a task that
// SendDeleteQuoteAuthorization queued, addressed to the author's followers and to the quoting
// actor, as Mastodon does.  An empty quoterURL addresses the followers alone.
func (service *Outbox) DeliverDeleteQuoteAuthorization(session data.Session, args mapof.Any, quoterURL string) {

	authorization := model.QuoteAuthorizationJSONLD(
		args.GetString("authorization"),
		args.GetString("attributedTo"),
		args.GetString("quote"),
		args.GetString("target"),
	)

	remove := deleteQuoteAuthorizationActivity(args.GetString("actor"), authorization, quoterURL, args.GetString("followers"))
	postcommit.Publish(session, service.queue, sender.OutboxSendToAllRecipients, remove)
}

// deleteQuoteAuthorizationActivity builds the "Delete" that DeliverDeleteQuoteAuthorization sends, in
// the shape of Mastodon's DeleteQuoteAuthorizationSerializer.
func deleteQuoteAuthorizationActivity(actorURL string, authorization mapof.Any, quoterURL string, followersURL string) mapof.Any {

	// The quoting actor is addressed only when the quote post still names one
	to := []string{vocab.NamespacePublic}

	if quoterURL != "" {
		to = append(to, quoterURL)
	}

	// The embedded object's type is what routes the Delete to Mastodon's revoke_quote, and an
	// embedded object never carries an @context of its own.
	object := maps.Clone(authorization)
	delete(object, vocab.AtContext)

	return mapof.Any{
		vocab.AtContext:      sliceof.Any{vocab.ContextTypeActivityStreams, vocab.ContextTypeQuotePosts, vocab.ContextTypeGoToSocial},
		vocab.PropertyID:     authorization.GetString(vocab.PropertyID) + "#delete",
		vocab.PropertyType:   vocab.ActivityTypeDelete,
		vocab.PropertyActor:  actorURL,
		vocab.PropertyObject: object,
		vocab.PropertyTo:     to,
		vocab.PropertyCC:     []string{followersURL},
	}
}
