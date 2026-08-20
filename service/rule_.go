package service

import (
	"context"
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/turbine/queue"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// actorLoader resolves a Fediverse address (a webfinger handle or a profile URL) to its canonical
// Actor document. *ActivityStream satisfies it; narrowing the Rule service's dependency to this one
// method keeps actor-trigger resolution unit-testable without standing up the full stream stack.
type actorLoader interface {
	GetActor(string) (streams.Document, error)
}

// Rule defines a service that manages all content rules created and imported by Users.
type Rule struct {
	activityStreamService  actorLoader
	importItemService      *ImportItem
	outboxService          *Outbox
	ruleSuppressionService *RuleSuppression
	userService            *User
	host                   string
	newSession             func(timeout time.Duration) (data.Session, context.CancelFunc, error)

	queue *queue.Queue
}

// NewRule returns a fully initialized Rule service
func NewRule() Rule {
	return Rule{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Rule) Refresh(factory *Factory) {
	service.activityStreamService = factory.ActivityStream()
	service.importItemService = factory.ImportItem()
	service.outboxService = factory.Outbox()
	service.ruleSuppressionService = factory.RuleSuppression()
	service.userService = factory.User()
	service.queue = factory.Queue()
	service.host = factory.Host()
	service.newSession = factory.Session
}

// Close stops any background processes controlled by this service
func (service *Rule) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the Rule collection for the provided database session
func (service *Rule) collection(session data.Session) data.Collection {
	return session.Collection("Rule")
}

// Count returns the number of Rule records that match the provided criteria
func (service *Rule) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an slice of allthe Rules that match the provided criteria
func (service *Rule) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {
	result := make([]model.Rule, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// QuerySummary returns an slice of allthe Rules that match the provided criteria
func (service *Rule) QuerySummary(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.RuleSummary, error) {
	result := make([]model.RuleSummary, 0)
	options = append(options, option.Fields(model.RuleSummaryFields()...))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Rules that match the provided criteria
func (service *Rule) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Rule records that match the provided criteria
func (service *Rule) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Rule], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Rule.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewRule), nil
}

// Load retrieves an Rule from the database
func (service *Rule) Load(session data.Session, criteria exp.Expression, rule *model.Rule) error {

	if err := service.collection(session).Load(notDeleted(criteria), rule); err != nil {
		return derp.Wrap(err, "service.Rule.Load", "Loading Rule", criteria)
	}

	return nil
}

// Save adds/updates an Rule in the database
func (service *Rule) Save(session data.Session, rule *model.Rule, note string) error {

	const location = "service.Rule.Save"

	// Validate the value before saving
	if _, err := service.Schema().Validate(rule); err != nil {
		return derp.Wrap(err, location, "Validating Rule", rule)
	}

	// Resolve the value the MatchKey is derived from -- WITHOUT touching the user-entered Trigger. A
	// hand-typed ACTOR Trigger may be a webfinger handle (@user@host) or an alias/redirecting profile
	// URL, neither of which equals the actor URL that inbound activities carry, so keying on it verbatim
	// would produce a rule that silently never matches. We resolve it to the canonical actor id for the
	// key only, leaving Trigger as the friendly value the user typed (for display). A Trigger that will
	// not resolve is REFUSED, not saved inert (a block that looks active but isn't is worse than a
	// visible error).
	matchKeyTrigger, err := service.resolveMatchKeyTrigger(rule)

	if err != nil {
		return derp.Wrap(err, location, "Resolving actor address", rule.Trigger)
	}

	// Recompute the derived match key from the (now-validated) Type and resolved trigger. UNCONDITIONAL:
	// the edit form posts Trigger, and a MatchKey that disagreed with it would silently stop matching --
	// a block that quietly stops blocking. Never sourced from a form (absent from RuleSchema).
	rule.MatchKey = model.RuleMatchKey(rule.Type, matchKeyTrigger)

	// Reconcile against any existing rule with the same (userId, matchKey): adopt its identity so this
	// save UPDATES that row in place rather than inserting a second row that would violate the unique
	// index. A real database error surfaces here (the old hasDuplicate silently dropped the write).
	if err := service.reconcileDuplicate(session, rule); err != nil {
		return derp.Wrap(err, location, "Reconciling duplicate Rule", rule)
	}

	// Snapshot the stored row (empty for a fresh insert) so the post-commit cleanup task can
	// detect transitions into/out of BLOCK and into MUTE (R8). Read AFTER reconcileDuplicate,
	// which may have re-pointed rule.RuleID at an adopted row.
	previous := service.previousRuleState(session, rule.RuleID)
	oldAction, oldMatchKey := previous.Action, previous.MatchKey

	// RULE: PublishedAction is server truth (P7-2) -- restored from the stored row so an inbound
	// record can neither forge nor erase the publish state. Stamped below on publish/retract.
	rule.PublishedAction = previous.PublishedAction

	// RULE: Externally imported rules cannot be re-shared automatically.
	if rule.OriginRemote() {
		rule.IsPublic = false
	}

	switch service.shouldPublish(*rule) {

	case true:

		switch rule.PublishDate {

		// "Publish" Rule when it is first shared publicly
		case 0:

			// Stamped BEFORE publishing because JSONLD reads PublishDate for the activity's
			// `published` property; publishing first would date every new activity to 1970.
			rule.PublishDate = time.Now().Unix()

			if err := service.publish(session, *rule); err != nil {
				return derp.Wrap(err, location, "Publishing Rule", rule)
			}

			// Record what actually went on the wire (P7-2)
			rule.PublishedAction = rule.Action

		// "Republish" changes when a public Rule is updated
		default:

			// republish retracts using the rule's PublishedAction (what the wire last saw),
			// then publishes the current Action -- so the stamp updates only afterwards (P7-2).
			if err := service.republish(session, *rule); err != nil {
				return derp.Wrap(err, location, "Republishing Rule", rule)
			}

			rule.PublishedAction = rule.Action
		}

	case false:

		// RULE: Retract Rules that still have a live published activity.
		// Keyed on PublishDate (the FACT that an activity is live) rather than IsPublic (the User's
		// INTENT): the two diverge for any Rule whose Action changed after it was published.
		if rule.PublishDate > 0 {

			// Retract BEFORE zeroing: the Undo embeds the original activity, whose `published` is
			// read from PublishDate. Zeroing first still finds the right OutboxMessage (that lookup
			// keys on the Rule's URL, not its date) while silently sending 1970 on the wire.
			if err := service.unpublish(session, *rule); err != nil {
				return derp.Wrap(err, location, "Unpublishing Rule", rule)
			}

			rule.PublishDate = 0
			rule.PublishedAction = ""
		}
	}

	// Save the rule to the database
	if err := service.collection(session).Save(rule, note); err != nil {
		return derp.Wrap(err, location, "Saving Rule", rule, note)
	}

	// Recalculate the rule count for this user
	// (skipped for domain-level Rules with no owning User)
	if err := service.userService.CalcRuleCount(session, rule.UserID); err != nil {
		return derp.Wrap(err, location, "Calculating rule count")
	}

	// Enqueue the retroactive cleanup task for action/trigger transitions (R8, post-commit)
	service.enqueueCleanup(session, *rule, oldAction, oldMatchKey, rule.Action)

	return nil
}

// resolveMatchKeyTrigger returns the value the Rule's MatchKey should be derived from, WITHOUT
// modifying the user-entered Trigger. For an ACTOR rule it loads the actor through the ActivityStream
// stack (which resolves webfinger @user@host handles and follows profile aliases/redirects) and
// returns its canonical id -- the exact value inbound activities carry, so the key matches. For other
// types (and empty triggers) it returns the Trigger unchanged. An ACTOR address that will not resolve
// to a real Actor is reported as a Validation error so the caller refuses the save rather than
// persisting a rule whose MatchKey can never match.
func (service *Rule) resolveMatchKeyTrigger(rule *model.Rule) (string, error) {

	// RULE: Only ACTOR triggers name an actor; DOMAIN (host) and TAG (token) resolve differently.
	if rule.Type != model.RuleTypeActor {
		return rule.Trigger, nil
	}

	// An empty trigger is a caller/validation concern, not something to network-resolve.
	if rule.Trigger == "" {
		return rule.Trigger, nil
	}

	// Key on the canonical id; the caller keeps rule.Trigger as the friendly value the user typed.
	return service.resolveActorAddress(rule.Trigger)
}

// resolveActorAddress resolves a Fediverse address -- a webfinger @user@host handle or an
// alias/redirecting profile URL -- to the canonical Actor id that ACTOR MatchKeys are derived from.
// GetActor canonicalizes both forms and fails when the address does not name a real Actor.
//
// Every caller that turns a user-friendly address into a MatchKey MUST route through here. Deriving
// a key from the raw address instead produces a key that no document can ever match (a rule that
// silently never fires), or -- for a lookup -- misses the rule it was searching for.
func (service *Rule) resolveActorAddress(address string) (string, error) {

	actor, err := service.activityStreamService.GetActor(address)

	if err != nil {
		// User-facing 422 whose message renders verbatim in the edit dialog, so it stays short.
		// The real cause (SSRF block, TLS/connection failure, 404, not-an-actor) rides along as a
		// detail so the error dump stays diagnosable -- otherwise every resolution failure
		// collapses to the same opaque message.
		return "", derp.Validation("Address not found", address, err)
	}

	return actor.ID(), nil
}

// Delete removes an Rule from the database (virtual delete)
func (service *Rule) Delete(session data.Session, rule *model.Rule, note string) error {

	const location = "service.Rule.Delete"

	// RULE: Retract the published activity for any Rule that has one.
	// Keyed on PublishDate (the FACT that an activity is live) rather than IsPublic (the User's
	// INTENT): the two diverge for any Rule whose Action changed after it was published. This runs
	// BEFORE the delete because callers on the Mastodon API hold no transaction, so retracting
	// afterwards would commit the delete and THEN report an error about a Rule that is already gone
	// -- which no retry could ever find again.
	if rule.PublishDate > 0 {
		if err := service.unpublish(session, *rule); err != nil {
			return derp.Wrap(err, location, "Unpublishing Rule", rule)
		}
	}

	// RULE: P7-3 -- deleting an IMPORTED rule writes a don't-re-import record FIRST, so the
	// provider's next backfill cannot resurrect it. (Locally-created rules have no RemoteID and
	// suppress nothing.) Written before the hard delete for the same reason retraction is: the
	// transactionless Mastodon-API path must not delete the row and THEN fail to remember why.
	if rule.RemoteID != "" {
		if err := service.ruleSuppressionService.Suppress(session, rule.UserID, rule.FollowingID, rule.RemoteID); err != nil {
			return derp.Wrap(err, location, "Suppressing re-import of deleted Rule", rule.RemoteID)
		}
	}

	// Hard-delete this Rule (D16). No tombstone remains -- so the unique {userId, matchKey} index has
	// a free slot to re-create the same rule, and cleanup tasks must carry a snapshot (they cannot load
	// the row after it is gone). The `note` is not journaled on a hard delete.
	if err := service.collection(session).HardDelete(exp.Equal("_id", rule.RuleID)); err != nil {
		return derp.Wrap(err, location, "Deleting Rule", rule)
	}

	// Recalculate the rule count for this user
	// (skipped for domain-level Rules with no owning User)
	if err := service.userService.CalcRuleCount(session, rule.UserID); err != nil {
		return derp.Wrap(err, location, "Calculating rule count")
	}

	// Enqueue the retroactive cleanup task -- deleting a BLOCK restores paused relationships (R8)
	service.enqueueCleanup(session, *rule, rule.Action, rule.MatchKey, "")

	// The Rule is gone, and so is its shadow on the wire.
	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Rule) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Rule record, without applying any additional business rules
func (service *Rule) HardDeleteByID(session data.Session, userID primitive.ObjectID, ruleID primitive.ObjectID) error {

	const location = "service.Rule.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", ruleID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting Rule", "userID: "+userID.Hex(), "ruleID: "+ruleID.Hex())
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Rule) ObjectType() string {
	return "Rule"
}

// New returns a fully initialized model.Rule as a data.Object.
func (service *Rule) ObjectNew() data.Object {
	result := model.NewRule()
	return &result
}

// ObjectID returns the unique ID of the provided Rule. Implements the ModelService interface.
func (service *Rule) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Rule); ok {
		return mention.RuleID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every Rule that matches the provided criteria. Implements the ModelService interface.
func (service *Rule) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single Rule as a data.Object. Implements the ModelService interface.
func (service *Rule) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewRule()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a Rule in the database. Implements the ModelService interface.
func (service *Rule) ObjectSave(session data.Session, object data.Object, comment string) error {
	if rule, ok := object.(*model.Rule); ok {
		return service.Save(session, rule, comment)
	}
	return derp.Internal("service.Rule.ObjectSave", "Invalid Object Type", object)
}

// ObjectDelete marks a Rule as deleted. Implements the ModelService interface.
func (service *Rule) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if rule, ok := object.(*model.Rule); ok {
		return service.Delete(session, rule, comment)
	}
	return derp.Internal("service.Rule.ObjectDelete", "Invalid Object Type", object)
}

// ObjectUserCan reports whether the provided Authorization may run an action on a Rule. Implements the ModelService interface.
func (service *Rule) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Rule", "Not Authorized")
}

// Schema returns the rosetta schema that describes a Rule
func (service *Rule) Schema() schema.Schema {
	return schema.New(model.RuleSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID retrieves a single Rule using its unique ID
func (service *Rule) LoadByID(session data.Session, userID primitive.ObjectID, ruleID primitive.ObjectID, rule *model.Rule) error {

	// RULE: UserID cannot be zero
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: RuleID cannot be zero
	if ruleID.IsZero() {
		return derp.Validation("RuleID cannot be zero")
	}

	criteria := exp.Equal("_id", ruleID).
		And(service.byUserID(userID))

	return service.Load(session, criteria, rule)
}

// LoadServerWideByID retrieves a single server-wide Rule — one owned by no User —
// for the admin console.  User-owned Rules are never returned.
func (service *Rule) LoadServerWideByID(session data.Session, ruleID primitive.ObjectID, rule *model.Rule) error {

	// RULE: RuleID cannot be zero
	if ruleID.IsZero() {
		return derp.Validation("RuleID cannot be zero")
	}

	criteria := exp.Equal("_id", ruleID).
		AndEqual("userId", primitive.NilObjectID)

	return service.Load(session, criteria, rule)
}

// LoadByToken retrieves a single Rule using its URL token
func (service *Rule) LoadByToken(session data.Session, userID primitive.ObjectID, token string, rule *model.Rule) error {

	// RULE: UserID cannot be zero
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: token must be a valid ObjectID
	ruleID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Rule.LoadByToken", "Converting token to ObjectID", token)
	}

	criteria := exp.Equal("_id", ruleID).AndEqual("userId", userID)

	return service.Load(session, criteria, rule)
}

// LoadByMatchKey retrieves the single Rule for the provided User and RuleType whose derived MatchKey
// equals RuleMatchKey(ruleType, trigger). The caller passes the CANONICAL trigger value -- an actor
// URL, Mastodon account id, or domain, NEVER a raw @user@host handle. The lookup keys on MatchKey
// (the rule's identity under the unique {userId, matchKey} index), NOT the stored Trigger, so it finds
// the rule regardless of the friendly value the user originally typed: a rule entered as "@alice@host"
// resolves to the same MatchKey as its canonical URL, so both are found by the same canonical lookup.
func (service *Rule) LoadByMatchKey(session data.Session, userID primitive.ObjectID, ruleType string, trigger string, rule *model.Rule) error {

	const location = "service.Rule.LoadByMatchKey"

	// RULE: UserID cannot be zero
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: RuleType cannot be empty
	if ruleType == "" {
		return derp.Validation("RuleType cannot be empty")
	}

	// RULE: Trigger cannot be empty
	if trigger == "" {
		return derp.Validation("Trigger cannot be empty")
	}

	// Derive the identity key the caller is looking for. An empty key (unknown type, or a trigger that
	// normalizes to nothing) can never identify a rule, so report Not Found rather than matching rows
	// that happen to carry an empty MatchKey.
	matchKey := model.RuleMatchKey(ruleType, trigger)

	if matchKey == "" {
		return derp.NotFound(location, "No rule matches the provided trigger", ruleType, trigger)
	}

	criteria := service.byUserID(userID).AndEqual("matchKey", matchKey)

	return service.Load(session, criteria, rule)
}

// LoadByFollowing retrieves a single Rule that maches the provided User, Following, RuleType, and Trigger
func (service *Rule) LoadByFollowing(session data.Session, userID primitive.ObjectID, followingID primitive.ObjectID, ruleType string, trigger string, rule *model.Rule) error {

	// RULE: UserID cannot be zero
	if userID.IsZero() {
		return derp.Validation("UserID cannot be zero")
	}

	// RULE: FollowingID cannot be zero
	if followingID.IsZero() {
		return derp.Validation("FollowingID cannot be zero")
	}

	// RULE: RuleType cannot be empty
	if ruleType == "" {
		return derp.Validation("RuleType cannot be empty")
	}

	// RULE: Trigger cannot be empty
	if trigger == "" {
		return derp.Validation("Trigger cannot be empty")
	}

	criteria := exp.Equal("userId", userID).
		AndEqual("type", ruleType).
		AndEqual("trigger", trigger).
		AndEqual("followingId", followingID)

	return service.Load(session, criteria, rule)
}

// QueryByType returns every public Rule of the provided type that belongs to a User
func (service *Rule) QueryByType(session data.Session, userID primitive.ObjectID, ruleType string, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {

	criteria = service.byUserID(userID).
		AndEqual("type", ruleType).
		AndEqual("isPublic", true).
		And(criteria)

	options = append(options, option.SortDesc("publishDate"))
	result, err := service.Query(session, criteria, options...)

	return result, err
}

// QueryByTypeDomain returns every public domain-blocking Rule that belongs to a User
func (service *Rule) QueryByTypeDomain(session data.Session, userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Rule, error) {
	return service.QueryByType(session, userID, model.RuleTypeDomain, criteria, options...)
}

// QueryByMatchKeys returns the RuleSummaries for a User (plus domain-wide rules) whose MatchKey is
// one of the provided keys -- typically model.DocumentMatchKeys(document). Every returned rule
// matches by construction, so the disposition engine only ranks them; it never re-scans. Expired
// rules are NOT filtered here (kept out of the query on purpose); the engine skips them by date.
func (service *Rule) QueryByMatchKeys(session data.Session, userID primitive.ObjectID, matchKeys []string) ([]model.RuleSummary, error) {

	// No keys means no possible match; skip the query entirely.
	if len(matchKeys) == 0 {
		return make([]model.RuleSummary, 0), nil
	}

	criteria := service.byUserID(userID).And(exp.In("matchKey", matchKeys))

	return service.QuerySummary(session, criteria)
}

// Disposition evaluates a document against this User's Rules (plus domain-wide admin rules) and
// returns the resulting RuleDisposition. `now` is the current Unix time in seconds.
func (service *Rule) Disposition(session data.Session, userID primitive.ObjectID, document streams.Document, now int64) (model.RuleDisposition, error) {

	// One matcher, one place (D17): Stage 1, Stage 2, and the future queue worker all adapt over this.
	// DocumentMatchKeys covers the actor AND the document's content tags.
	return service.DispositionForKeys(session, userID, model.DocumentMatchKeys(document), now)
}

// DispositionForKeys evaluates a pre-computed match-key set against this User's Rules (plus
// domain-wide admin rules). Passing NilObjectID as userID evaluates against admin-tier rules alone.
func (service *Rule) DispositionForKeys(session data.Session, userID primitive.ObjectID, keys []string, now int64) (model.RuleDisposition, error) {

	const location = "service.Rule.DispositionForKeys"

	// Keys, not a Document: the wire gate builds actor/domain keys by hand (claimed actor + keyId host).
	rules, err := service.QueryByMatchKeys(session, userID, keys)

	if err != nil {
		return model.RuleDisposition{}, derp.Wrap(err, location, "Querying rules by match key", userID, keys)
	}

	// Silence is golden
	return model.NewRuleDispositionForKeys(keys, rules, now), nil
}

// ActorDisposition evaluates the document's actor -- its ACTOR key and the DOMAIN keys of its host,
// but no content keys -- against this User's Rules (plus admin rules). `now` is the current Unix seconds.
func (service *Rule) ActorDisposition(session data.Session, userID primitive.ObjectID, document streams.Document, now int64) (model.RuleDisposition, error) {

	// Actor keys only; content/TAG filtering belongs to newsfeed ingest, not to "is this actor filtered?".
	return service.DispositionForKeys(session, userID, model.ActorMatchKeys(document.ActorID()), now)
}

// IsActorBlocked returns TRUE if the document's actor is BLOCK-ed for this User (plus admin-tier
// rules) by an ACTOR or DOMAIN rule. MUTE never gates the wire (D5), so only BLOCK counts.
func (service *Rule) IsActorBlocked(session data.Session, userID primitive.ObjectID, document streams.Document) (bool, error) {

	const location = "service.Rule.IsActorBlocked"

	// The Stage-2 gate and the Follow handlers share this so they agree on what "blocked" means.
	disposition, err := service.ActorDisposition(session, userID, document, time.Now().Unix())

	if err != nil {
		return false, derp.Wrap(err, location, "Checking block rules", document.ActorID())
	}

	// Not blocked? Then the dude abides.
	return disposition.IsBlocked(), nil
}

// DeliveryBlocked returns TRUE if outbound delivery from this User (or from the admin tier alone,
// for NilObjectID) to the recipient actor must be halted (R4). BLOCK-only: mute never gates
// egress (D5). Local and remote recipients are filtered identically (P5-1).
func (service *Rule) DeliveryBlocked(session data.Session, userID primitive.ObjectID, recipientURL string) bool {

	// An unidentifiable recipient cannot be cleared, so it is not delivered to
	if recipientURL == "" {
		return true
	}

	disposition, err := service.DispositionForKeys(session, userID, model.ActorMatchKeys(recipientURL), time.Now().Unix())

	// RULE: skip silently on error (P5-2) -- the recipient is treated as blocked, with no alert
	// and no retry. Wrongly delivering to a blocked actor is the unrecoverable failure; a dropped
	// delivery is indistinguishable from ordinary federation weather.
	if err != nil {
		return true
	}

	return disposition.IsBlocked()
}

// QueryDomainBlocks returns all external domains blocked by this Instance/Domain.
func (service *Rule) QueryDomainBlocks(session data.Session) ([]model.Rule, error) {

	criteria := exp.Equal("userId", primitive.NilObjectID).
		AndEqual("type", model.RuleTypeDomain).
		AndEqual("action", model.RuleActionBlock)

	return service.Query(session, criteria, option.SortAsc("trigger"))
}

// QueryBlockedActors returns all Actors blocked by this User (or by the Domain on behalf of the User)
func (service *Rule) QueryBlockedActors(session data.Session, userID primitive.ObjectID) ([]model.Rule, error) {

	criteria := service.byUserID(userID).
		AndEqual("type", model.RuleTypeActor).
		AndEqual("action", model.RuleActionBlock)

	return service.Query(session, criteria, option.SortAsc("trigger"))
}

// RangeByUserID returns all Rules tha belong to a specific User (NO DOMAIN RULES)
func (service *Rule) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.Rule], error) {
	return service.Range(session, exp.Equal("userId", userID))
}

// DeleteByUserID marks every Rule belonging to the provided User as deleted
func (service *Rule) DeleteByUserID(session data.Session, userID primitive.ObjectID, comment string) error {

	const location = "service.Rule.DeleteByUserID"

	rangeFunc, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Getting range function")
	}

	for rule := range rangeFunc {
		if err := service.Delete(session, &rule, comment); err != nil {
			return derp.Wrap(err, location, "Deleting rule", rule)
		}
	}

	return nil
}

/******************************************
 * Misc Helpers
 ******************************************/

// shouldPublish returns TRUE if this Rule federates to its Actor's followers.
func (service *Rule) shouldPublish(rule model.Rule) bool {

	// RULE: Users do not publish Rules. Federating moderation policy is a Domain act, so only
	// Domain-owned Rules are ever eligible (D9). A User's Rules are private, always.
	if !rule.OriginAdmin() {
		return false
	}

	// RULE: Only Rules that have been marked for sharing are published.
	if !rule.IsPublic {
		return false
	}

	// RULE: LABEL Rules do not federate.
	// Their only mapping was "Flag", which is a private moderation report everywhere else in the
	// Fediverse (R15), so labels stay local until the Notice store ships. This gate READS the Rule
	// and never writes it: deciding what a Rule means belongs to the caller.
	if rule.Action == model.RuleActionLabel {
		return false
	}

	return true
}

// reconcileDuplicate finds any OTHER rule with the same (userId, matchKey) and, if one exists, makes
// the incoming rule adopt its identity -- so Save updates that row IN PLACE instead of inserting a
// second row that would violate the unique {userId, matchKey} index. The incoming Action, ExpireDate,
// and other settings win (re-creating a rule over an expired one makes it active again -- D16), while
// the existing row's identity, provenance (FollowingID), publish state, and creation Journal are
// retained. Returns an error ONLY on a real database failure -- never swallowing it, unlike the old
// hasDuplicate, which returned TRUE on any non-NotFound error and silently dropped the write.
//
// IMPORTANT: this method MAY mutate the provided Rule.
func (service *Rule) reconcileDuplicate(session data.Session, rule *model.Rule) error {

	const location = "service.Rule.reconcileDuplicate"

	criteria := exp.NotEqual("_id", rule.RuleID).
		AndEqual("userId", rule.UserID).
		AndEqual("matchKey", rule.MatchKey)

	existing := model.NewRule()
	err := service.Load(session, criteria, &existing)

	// No existing rule with this key -- this save is a fresh insert.
	if derp.IsNotFound(err) {
		return nil
	}

	// A genuine database error must surface, not be swallowed.
	if err != nil {
		return derp.Wrap(err, location, "Loading possible duplicate", rule.MatchKey)
	}

	// Adopt the existing row so the save becomes an in-place update carrying the incoming settings.
	rule.RuleID = existing.RuleID
	rule.FollowingID = existing.FollowingID
	rule.PublishDate = existing.PublishDate
	rule.Journal = existing.Journal

	return nil
}

// byUserID generates a criteria expression that searches for:
// 1) Rules that belong to the provided User
// 2) Rules that belong to no User (i.e. domain-wide/admin rules)
//
// Expressed as a single IN (not an OR of equalities) so the query planner sees one bound set against
// the {userId, matchKey} index instead of an OR-of-ORs.
func (service *Rule) byUserID(userID primitive.ObjectID) exp.Expression {
	return exp.In("userId", []primitive.ObjectID{userID, primitive.NilObjectID})
}
