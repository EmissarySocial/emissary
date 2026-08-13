package activitypub

import (
	"net/http"

	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/validator"
	"github.com/rs/zerolog/log"
)

// SignatureVerifier verifies the HTTP Signature on an inbound request and returns the Signature that
// identifies the Actor who signed it.  service.ActivityStream.VerifySignature satisfies it.
type SignatureVerifier func(*http.Request) (sigs.Signature, error)

// HTTPSigValidator is a hannibal router.Validator that authenticates an inbound activity using HTTP
// Signatures.
type HTTPSigValidator struct {
	verifier SignatureVerifier // Verifies the request, and refreshes the signer's key when required
}

// NewHTTPSigValidator returns an HTTPSigValidator bound to a SignatureVerifier.
func NewHTTPSigValidator(verifier SignatureVerifier) HTTPSigValidator {

	// This replaces hannibal's validator.NewHTTPSig, which calls sigs.Verify exactly once and so has
	// no way to notice that a rejected signature was checked against a key the remote has since
	// rotated away from. The verifier passed in here owns that retry -- keeping the policy next to the
	// cache and the cooldown that bounds it, on the Emissary side. (BUG-22 D1)
	return HTTPSigValidator{
		verifier: verifier,
	}
}

// Validate returns ResultValid when the request carries a signature that verifies and speaks for the
// activity's own Actor.
func (validatorObject HTTPSigValidator) Validate(request *http.Request, activity *streams.Document) validator.Result {

	// An unsigned request is not this validator's to judge
	if !sigs.HasSignature(request) {
		return validator.ResultUnknown
	}

	// Verify the request against the signing Actor's public key
	signature, err := validatorObject.verifier(request)

	if err != nil {
		log.Trace().Err(err).Msg("Emissary Inbox: Error verifying HTTP Signature")
		return validator.ResultInvalid
	}

	// RULE: The Actor who owns the signature must be the Actor named in the Activity. Without this,
	// any peer holding a valid key of their own could deliver activities attributed to anyone.
	if signature.ActorID() != activity.Actor().ID() {
		log.Trace().Str("signatureActor", signature.ActorID()).Str("activityActor", activity.Actor().ID()).Msg("Emissary Inbox: HTTP Signature Actor does not match Activity Actor")
		return validator.ResultInvalid
	}

	// Come on in, the water's fine.
	return validator.ResultValid
}
