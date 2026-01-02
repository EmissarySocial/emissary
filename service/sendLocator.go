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
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/sherlock"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SendLocator is a service that locates Actors and Recipients for outbound ActivityPub messages.
type SendLocator struct {
	activityService      *ActivityStream
	encryptionKeyService *EncryptionKey
	followerService      *Follower
	userService          *User
	host                 string
	session              data.Session
}

// NewSendLocator returns a fully initialized SendLocator service
func NewSendLocator(factory *Factory, session data.Session) SendLocator {
	return SendLocator{
		activityService:      factory.ActivityStream(),
		encryptionKeyService: factory.EncryptionKey(),
		followerService:      factory.Follower(),
		userService:          factory.User(),
		host:                 factory.Host(),
		session:              session,
	}
}

// Actor is a part of the sender.Locator interface
// It returns an Actor interface for the provided Actor URL
func (service SendLocator) Actor(url string) (sender.Actor, error) {

	const location = "sender.SendLocator.Actor"

	// Parse the userID from the provided URL
	userID := service.ParseUserURI(url)

	if userID.IsZero() {
		return nil, derp.NotFound(location, "User not found", url)
	}

	// Load the User from the database
	user := model.NewUser()

	if err := service.userService.LoadByID(service.session, userID, &user); err != nil {
		return nil, derp.Wrap(err, location, "Unable to load user", "userID", userID.Hex())
	}

	// Load the User's Encryption Key
	encryptionKeyService := service.encryptionKeyService
	encryptionKey := model.NewEncryptionKey()

	if err := encryptionKeyService.LoadByParentID(service.session, model.EncryptionKeyTypeUser, user.UserID, &encryptionKey); err != nil {
		return nil, derp.Wrap(err, location, "Unable to load encryption key", "userID", user.UserID.Hex())
	}

	// Extract the Private Key
	privateKey, err := encryptionKeyService.GetPrivateKey(&encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to extract private key", "userID", user.UserID.Hex())
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

	// TODO: Special uri scheme for circle members
	// if strings.HasPrefix(uri, "circle:") {
	//	return service.resolveCircle(uri)
	// }

	// Special uri scheme for followers
	if userID := parseFollowersURI(service.host, uri); !userID.IsZero() {
		return service.resolveFollowers(userID)
	}

	// Special uri scheme for group members
	if strings.HasPrefix(uri, "group:") {
		return service.resolveGroup(uri)
	}

	// Otherwise, load the document at the provided URI/URL
	document, err := service.activityService.AppClient().Load(uri)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to load document for recipient", "uri", uri))
		return ranges.Empty[string](), nil
	}

	// Range over all documents in the collection
	if document.IsCollection() {
		return service.resolveCollection(document)
	}

	// Return the inbox URL for a single actor
	if document.IsActor() {
		return ranges.Values(document.PreferredInbox()), nil
	}

	// You suck. So you get nothing.
	return ranges.Empty[string](), nil
}

// Followers returns a RangeFunc with all inbox URLs for a followers uri
// This custom URI is used because followers may not be published in an ActivityPub collection
func (service SendLocator) resolveFollowers(userID primitive.ObjectID) (iter.Seq[string], error) {

	// Get all Followers for this User
	followers := service.followerService.RangeByUserID(service.session, userID)

	// Locate each Follower's inbox URL
	inboxURLs := ranges.Map(followers, func(follower model.Follower) string {
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
		return nil, derp.Wrap(err, location, "Unable to retrieve group members")
	}

	// Locate each Follower's inbox URL
	inboxURLs := ranges.Map(users, func(user model.User) string {
		return user.ActivityPubInboxURL()
	})

	// Success
	return inboxURLs, nil
}

// resolveCollection returns a RangeFunc with the inbox URLs for all actors contained in a collection
func (service SendLocator) resolveCollection(collection streams.Document) (iter.Seq[string], error) {

	// Get all documents in this collection
	documents := collections.RangeDocuments(collection)

	// Verify that documents are actors
	actors := ranges.Filter(documents, func(document streams.Document) bool {
		return document.IsActor()
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
		derp.Report(derp.Wrap(err, location, "Unable to load actor for inbox URL", "actorID", actorID))
		return ""
	}

	// Validate that this is actually an Actor
	if actor.NotActor() {
		return ""
	}

	// Retrurn the "best" inbox URL for this actor
	return actor.PreferredInbox()
}

// ParseUserURI parses user URIs in the format: https://<host>/@<userID>
// It returns the userID if successful, or primitive.NilObjectID if not.
func (service SendLocator) ParseUserURI(uri string) primitive.ObjectID {

	prefix := service.host + "/@"
	spew.Dump(uri, prefix)

	if strings.HasPrefix(uri, prefix) {
		token := strings.TrimPrefix(uri, prefix)
		spew.Dump(token)
		if userID, err := primitive.ObjectIDFromHex(token); err == nil {
			spew.Dump(userID)
			return userID
		}
	}

	// Nope
	spew.Dump("nope")
	return primitive.NilObjectID
}
