package service

import (
	"iter"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/rosetta/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * ActivityPub Methods
 ******************************************/

// ParseProfileURL parses (or looks up) the correct UserID from a given URL.
// Unlike the package-level ParseProfileURL, this method can resolve usernames into objectIDs
// because it has access to the database server.
func (service *User) ParseProfileURL(value string) (primitive.ObjectID, error) {

	const location = "service.User.ParseProfileURL"

	// Parse the URL to get the path
	urlValue, err := url.Parse(value)

	if err != nil {
		return primitive.NilObjectID, derp.Wrap(err, location, "Error parsing profile URL", value)
	}

	// RULE: server must be the same as the server we're running on
	if urlValue.Scheme+"://"+urlValue.Host != service.host {
		return primitive.NilObjectID, derp.BadRequestError(location, "Profile URL must exist on this server", urlValue, value, service.host)
	}

	// Extract the username from the URL
	path := list.BySlash(urlValue.Path).Tail()
	username := path.Head()

	if !strings.HasPrefix(username, "@") {
		return primitive.NilObjectID, derp.BadRequestError(location, "Username must begin with an '@'", value)
	}

	username = strings.TrimPrefix(username, "@")

	// If the username is already an objectID, then we can just return it.
	if userID, err := primitive.ObjectIDFromHex(username); err == nil {
		return userID, nil
	}

	// Otherwise, look it up in the database
	user := model.NewUser()

	if err := service.LoadByUsername(username, &user); err != nil {
		return primitive.NilObjectID, derp.Wrap(err, location, "Error loading user by username", username)
	}

	return user.UserID, nil
}

func (service *User) ActivityPubURL(userID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex()
}

func (service *User) ActivityPubPublicKeyURL(userID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex() + "#main-key" // was "/pub/key"
}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided User.
func (service *User) ActivityPubActor(userID primitive.ObjectID) (outbox.Actor, error) {

	const location = "service.Stream.ActivityPubActor"

	// Try to load the user's keys from the database
	encryptionKey := model.NewEncryptionKey()
	if err := service.keyService.LoadByParentID(model.EncryptionKeyTypeUser, userID, &encryptionKey); err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Error loading encryption key", userID)
	}

	// Extract the Private Key from the Encryption Key
	privateKey, err := service.keyService.GetPrivateKey(&encryptionKey)

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Error extracting private key", encryptionKey)
	}

	// Return the ActivityPub Actor
	actor := outbox.NewActor(
		service.ActivityPubURL(userID),
		privateKey,
		outbox.WithFollowers(service.rangeActivityPubFollowers(userID)),
		outbox.WithClient(service.activityStream),
		// TODO: Restore Queue:: , outbox.WithQueue(service.queue))
	)

	return actor, nil
}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided User.
func (service *User) rangeActivityPubFollowers(userID primitive.ObjectID) iter.Seq[string] {

	return func(yield func(string) bool) {

		followers := service.followerService.RangeActivityPubByType(model.FollowerTypeUser, userID)

		for follower := range followers {
			if !yield(follower.Actor.ProfileURL) {
				return
			}
		}
	}
}
