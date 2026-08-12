package service

import (
	"crypto"
	"iter"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * ActivityPub Methods
 ******************************************/

// ParseProfileURL parses (or looks up) the correct UserID from a given URL.
// Unlike the package-level ParseProfileURL, this method can resolve usernames into objectIDs
// because it has access to the database server.
func (service *User) ParseProfileURL(session data.Session, value string) (primitive.ObjectID, error) {

	const location = "service.User.ParseProfileURL"

	// Parse the URL to get the path
	parsedURL, err := url.Parse(value)

	if err != nil {
		return primitive.NilObjectID, derp.Wrap(err, location, "Parsing profile URL", value)
	}

	// RULE: server must be the same as the server we're running on
	if parsedURL.Scheme+"://"+parsedURL.Host != service.host {
		return primitive.NilObjectID, derp.BadRequest(location, "Profile URL must exist on this server", parsedURL, value, service.host)
	}

	// Extract the username from the URL
	path := list.BySlash(parsedURL.Path).Tail()
	username := path.Head()

	if !strings.HasPrefix(username, "@") {
		return primitive.NilObjectID, derp.BadRequest(location, "Username must begin with an '@'", value)
	}

	username = strings.TrimPrefix(username, "@")

	// If the username is already an objectID, then we can just return it.
	if userID, err := primitive.ObjectIDFromHex(username); err == nil {
		return userID, nil
	}

	// Otherwise, look it up in the database
	user := model.NewUser()

	if err := service.LoadByUsername(session, username, &user); err != nil {
		return primitive.NilObjectID, derp.Wrap(err, location, "Loading user by username", username)
	}

	return user.UserID, nil
}

// ActivityPubURL returns the canonical ActivityPub actor URL for the provided userID
func (service *User) ActivityPubURL(userID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex()
}

// PublicKeyID returns the key ID ("#main-key" fragment URL) for the provided userID
func (service *User) PublicKeyID(userID primitive.ObjectID) string {
	return service.ActivityPubURL(userID) + "#main-key"
}

// PrivateKey returns the signing key for the provided userID
func (service *User) PrivateKey(session data.Session, userID primitive.ObjectID) (crypto.PrivateKey, error) {

	const location = "service.User.PrivateKey"

	// Try to load the user's keys from the database
	encryptionKey := model.NewEncryptionKey()
	if err := service.keyService.LoadByParentID(session, model.EncryptionKeyTypeUser, userID, &encryptionKey); err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Loading encryption key", userID)
	}

	// Extract the Private Key from the Encryption Key
	privateKey, err := service.keyService.GetPrivateKey(&encryptionKey)

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Extracting private key", encryptionKey)
	}

	return privateKey, nil
}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided User.
func (service *User) ActivityPubActor(session data.Session, userID primitive.ObjectID) (outbox.Actor, error) {

	const location = "service.User.ActivityPubActor"

	// Extract the Private Key from the Encryption Key
	privateKey, err := service.PrivateKey(session, userID)

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Could not retrieve private key")
	}

	// Return the ActivityPub Actor
	actor := outbox.NewActor(
		service.ActivityPubURL(userID),
		privateKey,
		outbox.WithFollowers(service.rangeActivityPubFollowers(session, userID)),
		outbox.WithClient(service.activityService.UserClient(userID)),
		outbox.WithAllowPrivateIPs(service.activityService.AllowPrivateIPs()),
	)

	return actor, nil
}

// rangeActivityPubFollowers returns an iterator of profile URLs for the User's ActivityPub-method followers
func (service *User) rangeActivityPubFollowers(session data.Session, userID primitive.ObjectID) iter.Seq[string] {

	return func(yield func(string) bool) {

		followers := service.followerService.RangeActivityPubByType(session, model.FollowerTypeUser, userID)

		for follower := range followers {
			if !yield(follower.Actor.ProfileURL) {
				return
			}
		}
	}
}

// ActivityPubProfile returns the User's complete actor document: User.GetJSONLD() plus the
// publicKey block and, when the domain allows it, the MLS messaging properties.
func (service *User) ActivityPubProfile(session data.Session, user *model.User) (mapof.Any, error) {

	const location = "service.User.ActivityPubProfile"

	// Load the User's encryption key
	encryptionKey := model.NewEncryptionKey()

	if err := service.keyService.LoadByParentID(session, model.EncryptionKeyTypeUser, user.UserID, &encryptionKey); err != nil {
		return nil, derp.Wrap(err, location, "Loading encryption key", user.UserID)
	}

	// Combine the profile and the public key
	result := user.GetJSONLD()
	result[vocab.PropertyPublicKey] = mapof.Any{
		vocab.PropertyID:           user.ActivityPubPublicKeyURL(),
		vocab.PropertyOwner:        user.ActivityPubURL(),
		vocab.PropertyPublicKeyPEM: encryptionKey.PublicPEM,
	}

	// If the domain allows it, append MLS messaging values as well.
	if domain := service.domainService.Get(); domain.UserCanMLS(user) {
		result[vocab.PropertyMLSMessages] = user.ActivityPubInboxURL_DirectMessages_MLS()
		result[vocab.PropertyMLSKeyPackages] = user.ActivityPubKeyPackagesURL()
	}

	// Success!
	return result, nil
}

// sendProfileUpdate federates a changed profile: it wraps the User's complete actor document
// in an ActivityPub Update and hands it to the Outbox2 sender pipeline, record-less (see
// PROFILE-UPDATE-FEDERATION.md D-1). The fragment id derives from the profile fingerprint,
// so re-sends of the same profile state are idempotent for receivers that dedup by id.
func (service *User) sendProfileUpdate(session data.Session, user *model.User) error {

	const location = "service.User.sendProfileUpdate"

	// Assemble the complete actor document (profile + publicKey + MLS)
	object, err := service.ActivityPubProfile(session, user)

	if err != nil {
		return derp.Wrap(err, location, "Assembling actor document", user.UserID)
	}

	// Derive the activity's fragment id from the profile fingerprint
	fragment := user.ProfileFingerprint
	if len(fragment) > 16 {
		fragment = fragment[:16]
	}

	// Build the Update activity, addressed to this User's followers
	activity := mapof.Any{
		vocab.AtContext:      vocab.ContextTypeActivityStreams,
		vocab.PropertyID:     user.ActivityPubURL() + "#updates/" + fragment,
		vocab.PropertyType:   vocab.ActivityTypeUpdate,
		vocab.PropertyActor:  user.ActivityPubURL(),
		vocab.PropertyCC:     sliceof.String{user.ActivityPubFollowersURL()},
		vocab.PropertyObject: object,
	}

	// RULE: Only public profiles are also addressed to the Public audience
	if user.IsPublic {
		activity[vocab.PropertyTo] = sliceof.String{vocab.NamespaceActivityStreamsPublic}
	}

	// Deliver through the sender pipeline (post-commit)
	return service.outbox2Service.Send(session, activity)
}
