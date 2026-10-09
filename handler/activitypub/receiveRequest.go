package activitypub

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/assanitizer"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/router"
	"github.com/benpate/hannibal/streams"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ReceiveRequest parses an inbound ActivityPub request through the canonical inbox validator chain
// (Stage 1 of the block gate plus the standard validators), then strips reserved "emissary:"
// properties from the parsed activity. Pass NilObjectID as userID for admin-tier inboxes.
func ReceiveRequest(request *http.Request, client streams.Client, keys PublicKeyProvider, checker RuleChecker, session data.Session, userID primitive.ObjectID, options ...router.Option) (streams.Document, error) {

	const location = "handler.activitypub.ReceiveRequest"

	// ALL inbox families receive through this one funnel, so the validator chain and the sanitizer
	// cannot drift apart.

	// keys is a parameter rather than a router.Option because hannibal silently falls back to a
	// bare HTTP GET outside Emissary's client stack when its key finder is nil -- no ascache, no
	// asrules blocking, no AllowPrivateIPs policy. As an option it was opt-in, and three of the four
	// inboxes inherited that unprotected path (BUG-19).

	// The validator chain goes FIRST: caller options that patch the HTTPSig entry in place (as
	// router.WithPublicKeyFinder does) would be discarded if a later option replaced the chain.
	options = append([]router.Option{InboxValidators(keys, checker, session, userID)}, options...)

	// Receive and parse the activity
	activity, err := router.ReceiveRequest(request, client, options...)

	if err != nil {
		return activity, derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// RULE: An activity's id must share its actor's origin (D18). This closes the dedup-poisoning
	// primitive where an attacker pre-registers a victim's future activity id, and keeps a delivery
	// from filing itself under another host's id in the shared cache (BUG-223). A missing id cannot
	// poison, so it is exempt: the user inbox mints a local one, and the cache declines to store it.
	if activity.ID() != "" {
		if !activity.IsSameOrigin(activity.ActorID()) {
			return streams.NilDocument(), derp.Unauthorized(location, "Activity id must share the actor's origin", activity.ActorID(), activity.ID())
		}
	}

	// RULE: reserved "emissary:" properties are server-generated only, so inbound ones are
	// forgeries (fake moderation marks, fake trust annotations). Strip them here, before the
	// activity can reach storage, caches, notifications, or SSE payloads.
	assanitizer.Strip(activity.Value(), model.NamespaceEmissary)

	// So fresh and so clean, clean.
	return activity, nil
}
