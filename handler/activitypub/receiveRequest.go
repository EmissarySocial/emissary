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
// properties from the parsed activity. ALL inbox families receive through this one funnel, so the
// validator chain and the sanitizer cannot drift apart. Pass NilObjectID as userID for admin-tier
// inboxes.
func ReceiveRequest(request *http.Request, client streams.Client, checker RuleChecker, session data.Session, userID primitive.ObjectID, options ...router.Option) (streams.Document, error) {

	const location = "handler.activitypub.ReceiveRequest"

	// The validator chain goes FIRST: caller options like WithPublicKeyFinder patch the HTTPSig
	// entry in place, and would be discarded if a later option replaced the chain.
	options = append([]router.Option{InboxValidators(checker, session, userID)}, options...)

	// Receive and parse the activity
	activity, err := router.ReceiveRequest(request, client, options...)

	if err != nil {
		return activity, derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// RULE: reserved "emissary:" properties are server-generated only, so inbound ones are
	// forgeries (fake moderation marks, fake trust annotations). Strip them here, before the
	// activity can reach storage, caches, notifications, or SSE payloads.
	assanitizer.Strip(activity.Value(), model.NamespaceEmissary)

	// So fresh and so clean, clean.
	return activity, nil
}
