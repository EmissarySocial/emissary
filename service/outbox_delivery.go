package service

/******************************************
 * Follower Delivery
 *
 * The filter-and-dispatch loop that both outboxes share while
 * OUTBOX2-MIGRATION moves every writer onto Outbox2. Each recipient
 * passes the network boundary, the sender's rules, and the activity's
 * permissions, then receives it by its own delivery method.
 ******************************************/

import (
	"encoding/json"
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// followerEmailer sends an outbox activity to one email Follower
type followerEmailer interface {
	IsConfigured() bool
	SendFollowerActivity(follower *model.Follower, activity mapof.Any) error
}

// followerDelivery sends one activity to each recipient that passes the outbound rules
type followerDelivery struct {
	activityService *ActivityStream
	identityService *Identity
	ruleService     *Rule
	domainEmail     followerEmailer
	queue           *queue.Queue
	host            string
}

// deliver sends the activity, without its blind recipients, to every recipient the outbound rules allow
func (delivery followerDelivery) deliver(session data.Session, actorType string, actorID primitive.ObjectID, activity mapof.Any, permissions model.Permissions, recipients iter.Seq[model.Follower]) error {

	const location = "service.followerDelivery.deliver"

	// RULE: Deliver the stripped payload, never the original. Blind recipients still RECEIVE the
	// activity (the caller enumerated them from the full addressing) but must not SEE the list,
	// at any level of nesting.
	stripped, err := hannibal.StripBlindRecipients(activity)

	// RULE: An activity that cannot be stripped never will be, so this is a bad request, not a retry
	if err != nil {
		return derp.Wrap(err, location, "Removing blind recipients", derp.WithBadRequest())
	}

	payload := mapof.Any(stripped)

	// Serialize the payload once, so every ActivityPub delivery task shares the same string
	serialized, err := json.Marshal(payload)

	// RULE: An activity that cannot be serialized never will be, so this is a bad request, not a retry
	if err != nil {
		return derp.Wrap(err, location, "Serializing outbound activity", derp.WithBadRequest())
	}

	// The authoritative signer URL (stamped by Publish) — the delivery tasks resolve the key from it.
	actorURL := payload.GetString(vocab.PropertyActor)

	// The rules userID for the egress gate: a User actor binds their own rules; Stream and
	// SearchQuery actorIDs are NOT UserIDs, so those actors bind the admin tier alone.
	rulesUserID := ruleUserID(actorType, actorID)
	isLocalhost := uri.IsLocalHostname(delivery.host)

	// RULE: ask once, not once per Follower.  DomainEmail.Send now treats an unconfigured SMTP
	// connection as an error, and this loop can hold thousands of Followers -- reporting one
	// error each would flood the log on a domain that deliberately runs without email.
	canSendEmail := delivery.domainEmail.IsConfigured()

	if !canSendEmail {
		log.Debug().Str("location", location).Msg("SMTP is not configured. Skipping email Followers.")
	}

	// Filter and deliver to each recipient in turn
	for follower := range recipients {

		// Resolve the recipient's host from InboxURL, falling back to ProfileURL, because
		// addressees carry only a ProfileURL and would otherwise be dropped on a localhost domain (D8).
		recipientHost := follower.Actor.InboxURL
		if recipientHost == "" {
			recipientHost = follower.Actor.ProfileURL
		}

		// RULE: Only deliver to Followers on the same network as the Actor
		// (local can send to local, public can send to public, but local cannot send to public)
		if uri.IsLocalHostname(recipientHost) != isLocalhost {
			continue
		}

		// RULE: Delivery to blocked recipients is halted (R4) -- BOTH methods, email included
		if delivery.ruleService.DeliveryBlocked(session, rulesUserID, follower.Actor.ProfileURL) {
			log.Trace().Msg("Follower blocked by rule: " + follower.Actor.ProfileURL)
			continue
		}

		// RULE: Do not send to Followers who do not have permissions to view this activity
		if !delivery.identityService.HasPermissions(session, follower.Method, follower.Actor.ProfileURL, permissions) {
			log.Trace().Msg("Follower does not have permissions to view this activity: " + follower.Actor.ProfileURL)
			continue
		}

		log.Debug().Msg("Sending notification to Follower: " + follower.Actor.ProfileURL)

		// Deliver by the Follower's chosen method
		switch follower.Method {

		case model.FollowerMethodActivityPub:
			delivery.deliverActivityPub(session, actorURL, &follower, string(serialized))

		case model.FollowerMethodEmail:

			// RULE: Email only confirmed Followers. Anyone can type an address into the follow form,
			// so mailing an unconfirmed one defeats double opt-in (BUG-207 item 2).
			if !follower.IsActive() {
				continue
			}

			if canSendEmail {
				delivery.sendEmail(&follower, payload)
			}

		default:
			derp.Report(derp.Internal(location, "Unknown Follower Method.  This should never happen", follower))
		}
	}

	// Success!!
	return nil
}

// deliverActivityPub queues one retryable OutboxSendToSingleRecipient task that delivers the
// serialized activity to a single ActivityPub follower.
func (delivery followerDelivery) deliverActivityPub(session data.Session, actorURL string, follower *model.Follower, body string) {

	const location = "service.followerDelivery.deliverActivityPub"

	inboxURL := follower.Actor.InboxURL

	if inboxURL == "" {
		inboxURL = delivery.resolveInboxURL(follower.Actor.ProfileURL)
	}

	// A follower with no resolvable inbox cannot be delivered to. This matches the former
	// behavior (actor.SendOne failed and reported) — the delivery is dropped, not retried.
	if inboxURL == "" {
		derp.Report(derp.Internal(location, "Unable to resolve inbox URL for follower", follower.Actor.ProfileURL))
		return
	}

	postcommit.Publish(session, delivery.queue, sender.OutboxSendToSingleRecipient, mapof.Any{
		"actor": actorURL,
		"inbox": inboxURL,
		"body":  body,
	})
}

// sendEmail sends an email notification of the activity to a single "email" Follower
func (delivery followerDelivery) sendEmail(follower *model.Follower, activity mapof.Any) {

	const location = "service.followerDelivery.sendEmail"

	if err := delivery.domainEmail.SendFollowerActivity(follower, activity); err != nil {
		derp.Report(derp.Wrap(err, location, "Sending email", follower))
	}
}

// resolveInboxURL loads an Actor from its profile URL and returns its preferred inbox URL,
// or an empty string when the Actor cannot be loaded.
func (delivery followerDelivery) resolveInboxURL(profileURL string) string {

	const location = "service.followerDelivery.resolveInboxURL"

	if profileURL == "" {
		return ""
	}

	actor, err := delivery.activityService.AppClient().Load(profileURL)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading actor for inbox URL", profileURL))
		return ""
	}

	if actor.NotActor() {
		return ""
	}

	return actor.PreferredInbox()
}

// addresseesAsFollowers presents a list of addressee URLs as synthetic ActivityPub Followers, so that
// directly-addressed recipients can share the follower fan-out path
func addresseesAsFollowers(addressees iter.Seq[string]) iter.Seq[model.Follower] {

	return func(yield func(model.Follower) bool) {

		uniquer := streams.NewUniquer[string]()

		for addressee := range uniquer.Range(addressees) {
			follower := model.NewFollower()
			follower.Actor.ProfileURL = addressee
			follower.Method = model.FollowerMethodActivityPub
			follower.StateID = model.FollowerStateActive

			if !yield(follower) {
				return
			}
		}
	}
}
