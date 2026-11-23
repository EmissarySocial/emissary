package activitypub

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// SignatureParser resolves the Permissions and the Actor URL that a request's HTTP Signature grants
type SignatureParser interface {
	ParseHTTPSignatureWithActor(session data.Session, request *http.Request) (model.Permissions, string, error)
}

// VisibleOutboxItem returns an outbox item as JSON-LD when the request's signer may read it,
// and a NotFound error when they may not
func VisibleOutboxItem(parser SignatureParser, session data.Session, request *http.Request, item *model.OutboxItem) (mapof.Any, error) {

	const location = "handler.activitypub.VisibleOutboxItem"

	// RULE: A public item is served without verifying the signature, so a relay's subscribers can
	// re-fetch it without each costing a key lookup (OUTBOX2-OMNIBUS §1.23)
	if item.Permissions.IsAnonymous() {
		return item.GetJSONLD(), nil
	}

	// RULE: A signature that cannot be verified refuses the request. The refusal travels back
	// unwrapped, so the peer reads its fixed message.
	permissions, actorURL, err := parser.ParseHTTPSignatureWithActor(session, request)

	if err != nil {
		return nil, err
	}

	// RULE: A hidden item answers as a missing one, so a stranger cannot learn that it exists
	if !item.IsVisibleTo(permissions, actorURL) {
		return nil, derp.NotFound(location, "Activity not found", "activityId: "+item.ActivityID.Hex())
	}

	// Blind recipients are removed here
	return item.GetJSONLD(), nil
}
