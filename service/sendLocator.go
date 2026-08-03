package service

import (
	"iter"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/sherlock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SendLocator is a service that locates Actors and Recipients for outbound ActivityPub messages.
type SendLocator struct {
	activityService      *ActivityStream
	encryptionKeyService *EncryptionKey
	followerService      *Follower
	locatorService       *Locator
	userService          *User
	host                 string
	session              data.Session

	// deliveryBlocked is the R4 egress gate (normally Rule.DeliveryBlocked; a func field so tests
	// can inject verdicts). ruleUserID scopes it: a local User's own rules, or admin tier alone.
	deliveryBlocked func(data.Session, primitive.ObjectID, string) bool
	ruleUserID      primitive.ObjectID
}

// NewSendLocator returns a fully initialized SendLocator service
func NewSendLocator(factory *Factory, session data.Session) SendLocator {
	return SendLocator{
		activityService:      factory.ActivityStream(),
		encryptionKeyService: factory.EncryptionKey(),
		followerService:      factory.Follower(),
		locatorService:       factory.Locator(),
		userService:          factory.User(),
		host:                 factory.Host(),
		session:              session,
		deliveryBlocked:      factory.Rule().DeliveryBlocked,
		ruleUserID:           primitive.NilObjectID,
	}
}

// BoundToSender returns a copy of this SendLocator whose recipient resolution filters through the
// SENDING actor's block rules (R4). A local User actor binds their own rules; every other actor
// (Stream, SearchQuery, @search, @application, or an unparseable URL) binds the admin tier alone.
func (service SendLocator) BoundToSender(actorURL string) SendLocator {
	result := service
	result.ruleUserID = service.ParseUserURI(actorURL)
	return result
}

// recipientBlocked returns TRUE when delivery to this actor is halted by the sender's rules (R4).
// Local and remote recipients are filtered identically (P5-1).
func (service SendLocator) recipientBlocked(actorURL string) bool {
	return service.deliveryBlocked(service.session, service.ruleUserID, actorURL)
}

// Actor is a part of the sender.Locator interface. It returns a sender.Actor
// (signing key + public-key ID) for the provided LOCAL actor URL. It resolves
// every local actor type -- User, Stream, SearchQuery, the global @search actor,
// and the @application actor -- because Accepts, Announces, and Follows are sent
// by all of them, not just Users. See POST-COMMIT-FEDERATION.md F1.
func (service SendLocator) Actor(url string) (sender.Actor, error) {

	const location = "sender.SendLocator.Actor"

	// Fast path: local User actor. This branch is intentionally left byte-for-byte
	// identical to the original User-only implementation -- Users are the only actor
	// type exercised today, and this is signing code, so its behavior must not drift.
	if userID := service.ParseUserURI(url); !userID.IsZero() {
		return service.userActor(userID)
	}

	// Every other local actor type (Stream, SearchQuery, the global @search actor, and
	// @application) resolves its signing key through the shared Locator, which already
	// knows each type's key storage. actorTarget classifies the URL; GetPrivateKey loads
	// the key (Streams sign with a per-Stream key; the rest sign with the Domain key).
	actorType, actorID, ok := service.actorTarget(url)

	if !ok {
		return nil, derp.NotFound(location, "Actor not found", url)
	}

	publicKeyID, privateKey, err := service.locatorService.GetPrivateKey(service.session, actorType, actorID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading signing key", "url", url, "actorType", actorType)
	}

	// The activity's `actor` URL IS this actor's canonical ID, so reuse it directly.
	return sender.NewActor(url, publicKeyID, privateKey), nil
}

// actorTarget classifies a LOCAL actor URL into the (actorType, actorID) pair that
// Locator.GetPrivateKey consumes. ok is false when the URL is not a signable non-User
// local actor -- a User URL (Users take the fast path in Actor), an unrecognized
// objectType, or a path token that is not a valid ObjectID. It is a pure function of
// service.host and the URL (no database access) so the routing and ID-parsing decisions
// can be unit-tested independently of key storage. Stream actors are matched by their
// canonical hex-ID URL (host/<streamID>), which is the form the Outbox emits.
func (service SendLocator) actorTarget(url string) (actorType string, actorID primitive.ObjectID, ok bool) {

	resolvedType, token := locateObjectFromURL(service.host, url)

	switch resolvedType {
	case model.ActorTypeStream, model.ActorTypeSearchQuery,
		model.ActorTypeSearchDomain, model.ActorTypeApplication:
		// Signable non-User local actor.
	default:
		return "", primitive.NilObjectID, false
	}

	// Domain-level actors (@search, @application) carry no ID; GetPrivateKey ignores it.
	if token == "" {
		return resolvedType, primitive.NilObjectID, true
	}

	parsed, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return "", primitive.NilObjectID, false
	}

	return resolvedType, parsed, true
}

// userActor builds a sender.Actor for a local User. Extracted verbatim from the
// original Actor() implementation so the User signing path is unchanged.
func (service SendLocator) userActor(userID primitive.ObjectID) (sender.Actor, error) {

	const location = "sender.SendLocator.userActor"

	// Load the User from the database
	user := model.NewUser()

	if err := service.userService.LoadByID(service.session, userID, &user); err != nil {
		return nil, derp.Wrap(err, location, "Loading user", "userID", userID.Hex())
	}

	// Load the User's Encryption Key
	encryptionKey := model.NewEncryptionKey()

	if err := service.encryptionKeyService.LoadByParentID(service.session, model.EncryptionKeyTypeUser, user.UserID, &encryptionKey); err != nil {
		return nil, derp.Wrap(err, location, "Loading encryption key", "userID", user.UserID.Hex())
	}

	// Extract the Private Key
	privateKey, err := service.encryptionKeyService.GetPrivateKey(&encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "Extracting private key", "userID", user.UserID.Hex())
	}

	// Build an Actor object
	actor := sender.NewActor(
		user.ActivityPubURL(),
		user.ActivityPubPublicKeyURL(),
		privateKey,
	)

	// Success!
	return actor, nil
}

func (service SendLocator) Recipient(uri string) (iter.Seq[string], error) {

	const location = "sender.SendLocator.Recipient"

	// Skip empty URIs
	if uri == "" {
		return ranges.Empty[string](), nil
	}

	// RULE: The Public marker is an audience label, not a deliverable recipient. Skip it
	// explicitly -- otherwise it falls through to AppClient().Load() below, a wasted
	// outbound fetch (with a swallowed error) on every publicly-addressed send.
	if isPublicURI(uri) {
		return ranges.Empty[string](), nil
	}

	// RULE: Delivery to blocked recipients is halted (R4). Checking the raw URI first catches
	// blocked single actors AND anything on a blocked domain before a single byte is fetched from
	// them. Collection and follower URIs still expand below; their members are filtered one by one.
	if service.recipientBlocked(uri) {
		return ranges.Empty[string](), nil
	}

	// TODO: Special uri scheme for circle members
	// if strings.HasPrefix(uri, "circle:") {
	//	return service.resolveCircle(uri)
	// }

	// Special uri scheme for followers (User, Stream, or SearchQuery follower collections)
	if actorType, actorID := parseFollowersURI(service.host, uri); actorType != "" {
		return service.resolveFollowers(actorType, actorID)
	}

	// Special uri scheme for group members
	if strings.HasPrefix(uri, "group:") {
		return service.resolveGroup(uri)
	}

	// Otherwise, load the document at the provided URI/URL
	document, err := service.activityService.AppClient().Load(uri)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading document for recipient", "uri", uri))
		return ranges.Empty[string](), nil
	}

	// Range over all documents in the collection
	if document.IsCollection() {
		return service.resolveCollection(document)
	}

	// Return the inbox URL for a single actor
	if document.IsActor() {

		// RULE: Re-check the canonical actor ID -- redirects may make it differ from the URI (R4)
		if service.recipientBlocked(document.ID()) {
			return ranges.Empty[string](), nil
		}

		return ranges.Values(document.PreferredInbox()), nil
	}

	// You suck. So you get nothing.
	return ranges.Empty[string](), nil
}

// resolveFollowers returns a RangeFunc with the inbox URLs for an actor's followers collection.
// This custom URI is used because followers may not be published in an ActivityPub collection.
// It resolves the followers of ANY local actor type (User, Stream, SearchQuery) and considers
// ONLY ActivityPub-method followers — email/poll followers have no inbox, so they are not
// deliverable through the sender pipeline (they receive via the Outbox consumer's method dispatch).
func (service SendLocator) resolveFollowers(actorType string, actorID primitive.ObjectID) (iter.Seq[string], error) {

	// Get all ActivityPub Followers for this actor
	followers := service.followerService.RangeActivityPubByType(service.session, actorType, actorID)

	// RULE: Followers the sender blocked are stripped before inbox resolution (R4), so no fetch
	// is ever made on a blocked follower's behalf
	allowed := ranges.Filter(followers, func(follower model.Follower) bool {
		return !service.recipientBlocked(follower.Actor.ProfileURL)
	})

	// Locate each Follower's inbox URL
	inboxURLs := ranges.Map(allowed, func(follower model.Follower) string {
		return service.resolveInboxURL(follower.Actor.ProfileURL)
	})

	// Success
	return inboxURLs, nil
}

// resolveGroup returns a RangeFunc with the inbox URLs for all members of a group
// This custom URI is used because group members are not published in an ActivityPub collection
func (service SendLocator) resolveGroup(token string) (iter.Seq[string], error) {
	const location = "sender.SendLocator.Followers"

	// Get the userID from the provided token
	token = strings.TrimPrefix(token, "group:")
	groupID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return nil, derp.Wrap(err, location, "Invalid user ID", "token: "+token)
	}

	// Get all members of this Group
	users, err := service.userService.RangeByGroup(service.session, groupID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Retrieving group members")
	}

	// RULE: Local recipients are subject to the same rule filters as remote ones (P5-1)
	allowed := ranges.Filter(users, func(user model.User) bool {
		return !service.recipientBlocked(user.ActivityPubURL())
	})

	// Locate each Follower's inbox URL
	inboxURLs := ranges.Map(allowed, func(user model.User) string {
		return user.ActivityPubInboxURL()
	})

	// Success
	return inboxURLs, nil
}

// resolveCollection returns a RangeFunc with the inbox URLs for all actors contained in a collection
func (service SendLocator) resolveCollection(collection streams.Document) (iter.Seq[string], error) {

	// Get all documents in this collection
	documents := collections.RangeDocuments(collection)

	// Verify that documents are actors, and strip any the sender has blocked (R4)
	actors := ranges.Filter(documents, func(document streams.Document) bool {
		return document.IsActor() && !service.recipientBlocked(document.ID())
	})

	// Find the best inbox URL for each document
	inboxURLs := ranges.Map(actors, func(document streams.Document) string {
		return document.LoadLink().PreferredInbox()
	})

	// Done.
	return inboxURLs, nil
}

// resolveInboxURL loads an Actor and returns the best inbox URL for a specific actorID
func (service SendLocator) resolveInboxURL(actorID string) string {

	const location = "sender.SendLocator.resolveInboxURL"

	// Retrieve the Actor document from the ActivityPub client
	actor, err := service.activityService.AppClient().Load(actorID, sherlock.AsActor())

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading actor for inbox URL", "actorID", actorID))
		return ""
	}

	// Validate that this is actually an Actor
	if actor.NotActor() {
		return ""
	}

	// Retrurn the "best" inbox URL for this actor
	return actor.PreferredInbox()
}

// isPublicURI returns TRUE when the URI is the ActivityStreams "Public" audience marker,
// in any of its three accepted AS2 forms (full URI, "as:Public", bare "Public").
func isPublicURI(uri string) bool {

	switch uri {
	case vocab.NamespaceActivityStreamsPublic, vocab.NamespaceASPublic, vocab.NamespacePublic:
		return true
	}

	return false
}

// ParseUserURI parses user URIs in the format: https://<host>/@<userID>
// It returns the userID if successful, or primitive.NilObjectID if not.
func (service SendLocator) ParseUserURI(uri string) primitive.ObjectID {

	prefix := service.host + "/@"

	if strings.HasPrefix(uri, prefix) {
		token := strings.TrimPrefix(uri, prefix)
		if userID, err := primitive.ObjectIDFromHex(token); err == nil {
			return userID
		}
	}

	// Nope
	return primitive.NilObjectID
}
