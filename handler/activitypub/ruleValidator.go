package activitypub

import (
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/router"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/validator"
	"github.com/benpate/hannibal/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InboxValidators returns the router Option that installs the canonical inbox validator chain (Stage 1
// of the block gate plus the standard validators). Pass NilObjectID as userID for admin-tier inboxes.
func InboxValidators(keyFinder sigs.PublicKeyFinder, checker RuleChecker, session data.Session, userID primitive.ObjectID) router.Option {

	// keyFinder is required, not optional: a nil one sends hannibal down its unprotected fallback
	// path. See ReceiveRequest for the full reasoning (BUG-19).

	// One definition so the chain cannot drift: WithValidators REPLACES it wholesale, so hand-assembling
	// it per handler risks omitting NewHTTPSig and silently disabling signature verification there.
	return router.WithValidators(
		NewRuleValidator(checker, session, userID),
		validator.NewHTTPSig(keyFinder),
		validator.NewDeletedObject(),
	)
}

// RuleChecker is the subset of the Rule service the inbox gate needs: an indexed lookup that evaluates
// match keys against a User's Rules. *service.Rule satisfies it.
type RuleChecker interface {
	DispositionForKeys(session data.Session, userID primitive.ObjectID, keys []string, now int64) (model.RuleDisposition, error)
}

// RuleValidator is Stage 1 of the inbox block gate: a hannibal router.Validator that discards
// activities from blocked origins before signature verification.
type RuleValidator struct {
	checker RuleChecker        // Evaluates match keys against the inbox owner's Rules
	session data.Session       // Closed over at construction, because the validator is built per-request
	userID  primitive.ObjectID // Inbox owner, or NilObjectID for the admin-tier inboxes
}

// NewRuleValidator returns a RuleValidator bound to a checker, session, and inbox-owner UserID.
func NewRuleValidator(checker RuleChecker, session data.Session, userID primitive.ObjectID) RuleValidator {
	return RuleValidator{
		checker: checker,
		session: session,
		userID:  userID,
	}
}

// IsWireGateException returns TRUE for the activity types the inbox block gate verifies but never
// fast-discards: Follow, Delete, Undo, and Move.
func IsWireGateException(activityType string) bool {

	// These four are the D5 exception set. Their handlers deliberately process a blocked actor's
	// activity (loud Follow reject; subtractive Delete/Undo per D6; Move copies the block per R20), so
	// discarding here would make that unreachable.
	//
	// Why Follow may reject LOUDLY (403) while everything else stays silent (401): the loud response
	// fires only AFTER signature verification, so a caller can only ever test identities it owns
	// ("am I blocked?") -- never probe the block list for third parties. And Follow reveals that answer
	// anyway: the protocol completes with an Accept, so a silently-dropped Follow is an eternally
	// pending request -- a tell in itself, plus dangling state on both servers. Other activities expect
	// no reply, so for them silence is genuinely indistinguishable and leaks nothing.
	switch activityType {
	case vocab.ActivityTypeFollow, vocab.ActivityTypeDelete, vocab.ActivityTypeUndo, vocab.ActivityTypeMove:
		return true
	}
	return false
}

// Validate runs the Stage-1 check on the CLAIMED (unverified) actor and signature keyId. It returns
// ResultInvalid to discard a blocked origin, or ResultUnknown to defer, and never ResultValid.
func (v RuleValidator) Validate(request *http.Request, document *streams.Document) validator.Result {

	// It may DENY but never GRANT: a ResultValid would short-circuit the chain (validateRequest is
	// first-decisive-wins) and skip signature verification entirely.

	// Exception-set types are verified first, then handled (D5), so they are never discarded here.
	if IsWireGateException(document.Type()) {
		return validator.ResultUnknown
	}

	// RULE: inline non-public MLS is never fast-discarded (4B): every ciphertext must reach storage
	// for epoch safety, so its rules are evaluated -- and stamped -- at Stage 2 instead. No possible
	// discard means no reason to query, either.
	if IsMLSCreate(*document) {
		return validator.ResultUnknown
	}

	// Keys from the claimed actor (ACTOR + its domain suffixes) plus the signature keyId host (as
	// DOMAIN keys), so a DOMAIN block extends to the delivering server and its key is never fetched.
	keys := model.ActorMatchKeys(document.ActorID())
	keys = append(keys, keyIDDomainKeys(request)...)

	disposition, err := v.checker.DispositionForKeys(v.session, v.userID, keys, time.Now().Unix())

	// Fail OPEN to Stage 2 (D17): Stage 1 is an optimization layer; a DB blip must not break all
	// federation. The authoritative Stage-2 check still runs after verification, and it fails closed.
	if err != nil {
		return validator.ResultUnknown
	}

	// A blocked origin is discarded pre-verification. hannibal surfaces ResultInvalid as the same 401
	// a signature failure returns (D3), so a blocked server cannot enumerate the block list.
	if disposition.IsBlocked() {
		return validator.ResultInvalid
	}

	// Not blocked (or only muted -- MUTE never gates the wire, D5). Defer to signature verification.
	return validator.ResultUnknown
}

// keyIDDomainKeys returns the DOMAIN match keys of the signature keyId's host, or nil if the request is
// unsigned or the signature will not parse.
func keyIDDomainKeys(request *http.Request) []string {

	// The keyId names the SERVER delivering this activity, so binding it lets a DOMAIN block refuse a
	// blocked relay before its key is ever fetched.
	signature := sigs.GetSignature(request)

	if signature == "" {
		return nil
	}

	parsed, err := sigs.ParseSignature(signature)

	if err != nil {
		return nil
	}

	return model.DomainMatchKeys(parsed.ActorID())
}
