package service

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/rosetta/iterator"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User manages all interactions with the User collection
type User struct {
	collection    data.Collection
	followers     data.Collection
	following     data.Collection
	blocks        data.Collection
	streamService *Stream
	emailService  *DomainEmail
	folderService *Folder
	keyService    *EncryptionKey
	host          string
}

// NewUser returns a fully populated User service
func NewUser() User {
	return User{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *User) Refresh(userCollection data.Collection, followerCollection data.Collection, followingCollection data.Collection, blockCollection data.Collection, streamService *Stream, keyService *EncryptionKey, emailService *DomainEmail, folderService *Folder, host string) {
	service.collection = userCollection
	service.followers = followerCollection
	service.following = followingCollection
	service.blocks = blockCollection

	service.streamService = streamService
	service.emailService = emailService
	service.folderService = folderService
	service.keyService = keyService

	service.host = host
}

// Close stops any background processes controlled by this service
func (service *User) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// List returns an iterator containing all of the Users who match the provided criteria
func (service *User) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an User from the database
func (service *User) Load(criteria exp.Expression, result *model.User) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.User.Load", "Error loading User", criteria)
	}

	return nil
}

// Save adds/updates an User in the database
func (service *User) Save(user *model.User, note string) error {

	isNew := user.IsNew()

	// RULE: Set ProfileURL to the hostname + the username
	user.ProfileURL = service.host + "/@" + user.UserID.Hex()

	// RULE: If password reset has already expired, then clear the reset code
	if (user.PasswordReset.ExpireDate > 0) && (user.PasswordReset.ExpireDate < time.Now().Unix()) {
		user.PasswordReset.AuthCode = ""
	}

	// Clean the value before saving
	if err := service.Schema().Clean(user); err != nil {
		return derp.Wrap(err, "service.User.Save", "Error cleaning User", user)
	}

	// Try to save the User record to the database
	if err := service.collection.Save(user, note); err != nil {
		return derp.Wrap(err, "service.User.Save", "Error saving User", user, note)
	}

	// RULE: Take these actions when setting up a new user
	if isNew {

		// RULE: Create a new encryption key for this user
		if _, err := service.keyService.Create(user.UserID); err != nil {
			return derp.Wrap(err, "service.User.Save", "Error creating encryption key for User", user, note)
		}

		// RULE: Create default folders for this user
		if err := service.folderService.CreateDefaultFolders(user.UserID); err != nil {
			return derp.Wrap(err, "service.User.Save", "Error creating default folders for User", user, note)
		}
	}

	// RULE: If the user has not yet been sent their password, then try to send it now.
	if user.PasswordReset.CreateDate == 0 {
		service.SendWelcomeEmail(user)
	}

	// Success!
	return nil
}

// Delete removes an User from the database (virtual delete)
func (service *User) Delete(user *model.User, note string) error {

	if err := service.collection.Delete(user, note); err != nil {
		return derp.Wrap(err, "service.User.Delete", "Error deleting User", user, note)
	}

	// TODO: HIGH: Clean up related records (like InboxItem and OutboxItem)
	if err := service.streamService.DeleteByParent(user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, "service.User.Delete", "Error deleting User's streams", user, note)
	}

	return nil
}

/******************************************
 * Generic Data Functions
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *User) ObjectType() string {
	return "User"
}

// New returns a fully initialized model.Stream as a data.Object.
func (service *User) ObjectNew() data.Object {
	result := model.NewUser()
	return &result
}

func (service *User) ObjectID(object data.Object) primitive.ObjectID {

	if user, ok := object.(*model.User); ok {
		return user.UserID
	}

	return primitive.NilObjectID
}

func (service *User) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *User) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *User) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewUser()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *User) ObjectSave(object data.Object, note string) error {
	if user, ok := object.(*model.User); ok {
		return service.Save(user, note)
	}
	return derp.NewInternalError("service.User.ObjectSave", "Invalid object type", object)
}

func (service *User) ObjectDelete(object data.Object, note string) error {
	if user, ok := object.(*model.User); ok {
		return service.Delete(user, note)
	}
	return derp.NewInternalError("service.User.ObjectDelete", "Invalid object type", object)
}

func (service *User) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.User.ObjectUserCan", "Not Authorized")
}

func (service *User) Schema() schema.Schema {
	return schema.New(model.UserSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *User) ListUsernameOrOwner(username string) (data.Iterator, error) {
	return service.List(exp.Equal("isOwner", true).OrEqual("username", username))
}

func (service *User) ListOwners() (data.Iterator, error) {
	return service.List(exp.Equal("isOwner", true))
}

func (service *User) ListOwnersAsSlice() []model.UserSummary {
	it, _ := service.ListOwners()
	return iterator.Slice(it, model.NewUserSummary)
}

// ListByIdentities returns all users that appear in the list of identities
func (service *User) ListByIdentities(identities []string) (data.Iterator, error) {
	return service.List(exp.In("identities", identities))
}

// ListByGroup returns all users that match a provided group name
func (service *User) ListByGroup(group string) (data.Iterator, error) {
	return service.List(exp.Equal("groupId", group))
}

// LoadByID loads a single model.User object that matches the provided userID
func (service *User) LoadByID(userID primitive.ObjectID, result *model.User) error {
	criteria := exp.Equal("_id", userID)
	return service.Load(criteria, result)
}

func (service *User) LoadByProfileURL(profileUrl string, result *model.User) error {
	criteria := exp.Equal("profileUrl", profileUrl)
	return service.Load(criteria, result)
}

// LoadByUsername loads a single model.User object that matches the provided username
func (service *User) LoadByUsername(username string, result *model.User) error {
	criteria := exp.Equal("username", username)
	return service.Load(criteria, result)
}

// LoadByUsernameOrEmail loads a single model.User object that matches the provided username
func (service *User) LoadByUsernameOrEmail(usernameOrEmail string, result *model.User) error {
	criteria := exp.Equal("username", usernameOrEmail).OrEqual("emailAddress", usernameOrEmail)
	err := service.Load(criteria, result)

	return err
}

// LoadByUsername loads a single model.User object that matches the provided token
func (service *User) LoadByToken(token string, result *model.User) error {

	// If the token *looks* like an ObjectID then try that first.  If it works, then return in triumph
	if userID, err := primitive.ObjectIDFromHex(token); err == nil {
		if err := service.LoadByID(userID, result); err == nil {
			return nil
		}
	}

	// Otherwise, use the token as a username
	return service.LoadByUsername(token, result)
}

func (service *User) LoadByResetCode(userID string, code string, user *model.User) error {

	// Try to find the user by ID
	if err := service.LoadByToken(userID, user); err != nil {
		return derp.Wrap(err, "service.User.LoadByResetCode", "Error loading User by ID", userID)
	}

	// If the password reset is not valid, then return an "Unauthorized" error
	if !user.PasswordReset.IsValid(code) {
		return derp.NewUnauthorizedError("service.User.LoadByResetCode", "Invalid password reset code", userID, code)
	}

	// No Error means success
	return nil
}

// Count returns the number of (non-deleted) records in the User collection
func (service *User) Count(ctx context.Context, criteria exp.Expression) (int, error) {
	return queries.CountRecords(ctx, service.collection, notDeleted(criteria))
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *User) CalcFollowerCount(userID primitive.ObjectID) error {
	err := queries.SetFollowersCount(service.collection, service.followers, userID)
	return derp.Wrap(err, "service.User.CalcFollowerCount", "Error setting follower count", userID)
}

func (service *User) CalcFollowingCount(userID primitive.ObjectID) error {
	err := queries.SetFollowingCount(service.collection, service.following, userID)
	return derp.Wrap(err, "service.User.CalcFollowingCount", "Error setting following count", userID)
}

func (service *User) CalcBlockCount(userID primitive.ObjectID) {
	err := queries.SetBlockCount(service.collection, service.blocks, userID)
	// nolint: errcheck
	derp.Report(derp.Wrap(err, "service.User.CalcBlockCount", "Error setting block count", userID))
}

func (service *User) SetOwner(owner config.Owner) error {

	// Try to read the owner from the database
	users, err := service.ListUsernameOrOwner(owner.Username)

	if err != nil {
		return derp.Wrap(err, "service.User.SetOwner", "Error loading owners")
	}

	user := model.NewUser()
	found := false

	for users.Next(&user) {

		// See if this user is the "owner" being added/updated
		isOwner := (user.Username == owner.Username)

		// Mark "found" if possible
		if isOwner {
			found = true
		}

		// If we're changing this record, then save it.
		if user.IsOwner != isOwner {
			user.IsOwner = isOwner

			if err := service.Save(&user, "AssertOwner"); err != nil {
				return derp.Wrap(err, "service.User.SetOwner", "Error saving user", user)
			}
		}

		// Reset the user object
		user = model.NewUser()
	}

	// If we didn't find an owner above, then we need to create one.
	if !found {
		user := model.NewUser()
		user.DisplayName = owner.DisplayName
		user.EmailAddress = owner.EmailAddress
		user.Username = owner.Username
		user.IsOwner = true

		if err := service.Save(&user, "CreateOwner"); err != nil {
			return derp.Wrap(err, "service.User.SetOwner", "Error saving user", user)
		}
	}

	return nil
}

// MakeNewPasswordResetCode generates a new password reset code for the provided user.
func (service *User) MakeNewPasswordResetCode(user *model.User) error {

	// Create a new password reset code for this user
	user.PasswordReset = model.NewPasswordReset(24 * time.Hour)

	// Try to save the user with the new password reset code.
	if err := service.Save(user, "Create Password Reset Code"); err != nil {

		return derp.Wrap(err, "service.User.MakeNewPasswordResetCode", "Error saving user", user)
	}

	return nil
}

// ResetPassword resets the password for the provided user
func (service *User) ResetPassword(user *model.User, resetCode string, newPassword string) error {

	// Verify that the password reset code is valid
	if !user.PasswordReset.IsValid(resetCode) {
		return derp.NewForbiddenError("service.User.ResetPassword", "Invalid password reset code", user, resetCode)
	}

	// Update the password
	user.Password = newPassword

	// Invalidate the old reset code.
	user.PasswordReset = model.PasswordReset{}

	// Try to save the user with the new password reset code.
	if err := service.Save(user, "Create Password Reset Code"); err != nil {
		return derp.Wrap(err, "service.User.ResetPassword", "Error saving user", user)
	}

	return nil
}

// SendWelcomeEmail generates a new password reset code and sends a welcome email to a new user.
// If there is a problem sending the email, then the new code is not saved.
func (service *User) SendWelcomeEmail(user *model.User) {

	if err := service.MakeNewPasswordResetCode(user); err != nil {
		derp.Report(derp.Wrap(err, "service.User.SendWelcomeEmail", "Error making password reset", user))
		return
	}

	// Try to send the welcome email.  If it fails, then don't save the new password reset code.
	if err := service.emailService.SendWelcome(user); err != nil {
		derp.Report(derp.Wrap(err, "service.User.SendWelcomeEmail", "Error sending welcome email", user))
		return
	}
}

// SendPasswordResetEmail generates a new password reset code and sends a welcome email to a new user.
// If there is a problem sending the email, then the new code is not saved.
func (service *User) SendPasswordResetEmail(user *model.User) {

	if err := service.MakeNewPasswordResetCode(user); err != nil {
		derp.Report(derp.Wrap(err, "service.User.SendPasswordResetEmail", "Error making password reset", user))
		return
	}

	// Try to send the welcome email.  If it fails, then don't save the new password reset code.
	if err := service.emailService.SendPasswordReset(user); err != nil {
		derp.Report(derp.Wrap(err, "service.User.SendPasswordResetEmail", "Error sending password reset", user))
		return
	}
}

/******************************************
 * ActivityPub Methods
 ******************************************/

// ParseProfileURL parses (or looks up) the correct UserID from a given URL.
// Unlike the package-level ParseProfileURL, this method can resolve usernames into objectIDs
// because it has access to the database server.
func (service *User) ParseProfileURL(value string) (primitive.ObjectID, error) {

	// Parse the URL to get the path
	urlValue, err := url.Parse(value)

	if err != nil {
		return primitive.NilObjectID, derp.Wrap(err, "service.User.ParseProfileURL", "Error parsing profile URL", value)
	}

	// RULE: server must be the same as the server we're running on
	if urlValue.Scheme+"://"+urlValue.Host != service.host {
		return primitive.NilObjectID, derp.New(derp.CodeBadRequestError, "service.User.ParseProfileURL", "Profile URL must exist on this server", urlValue, value, service.host)
	}

	// Extract the username from the URL
	path := list.BySlash(urlValue.Path).Tail()
	username := path.Head()

	if !strings.HasPrefix(username, "@") {
		return primitive.NilObjectID, derp.New(derp.CodeBadRequestError, "service.User.ParseProfileURL", "Username must begin with an '@'", value)
	}

	username = strings.TrimPrefix(username, "@")

	// If the username is already an objectID, then we can just return it.
	if userID, err := primitive.ObjectIDFromHex(username); err == nil {
		return userID, nil
	}

	// Otherwise, look it up in the database
	user := model.NewUser()

	if err := service.LoadByUsername(username, &user); err != nil {
		return primitive.NilObjectID, derp.Wrap(err, "service.User.ParseProfileURL", "Error loading user by username", username)
	}

	return user.UserID, nil
}

func (service *User) ActivityPubURL(userID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex()
}

func (service *User) ActivityPubPublicKeyURL(userID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex() + "/pub/key"
}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided user.
func (service *User) ActivityPubActor(userID primitive.ObjectID) (pub.Actor, error) {

	// Try to load the user's keys from the database
	encryptionKey := model.NewEncryptionKey()
	if err := service.keyService.LoadByID(userID, &encryptionKey); err != nil {
		return pub.Actor{}, derp.Wrap(err, "service.Following.ActivityPubActor", "Error loading encryption key", userID)
	}

	// Extract the Private Key from the Encryption Key
	privateKey, err := service.keyService.GetPrivateKey(&encryptionKey)

	if err != nil {
		return pub.Actor{}, derp.Wrap(err, "service.Following.ActivityPubActor", "Error extracting private key", encryptionKey)
	}

	// Return the ActivityPub Actor
	return pub.Actor{
		ActorID:     service.ActivityPubURL(userID),
		PublicKeyID: service.ActivityPubPublicKeyURL(userID),
		PublicKey:   privateKey.PublicKey,
		PrivateKey:  privateKey,
	}, nil
}

/******************************************
 * WebFinger Behavior
 ******************************************/

func (service *User) LoadWebFinger(username string) (digit.Resource, error) {

	switch {

	case domain.HasProtocol(username):
		username = list.Last(username, '@')
		username = list.First(username, '/')

	case strings.HasPrefix(username, "acct:"):
		// Trim prefixes "acct:" and "@"
		username = strings.TrimPrefix(username, "acct:")
		username = strings.TrimPrefix(username, "@")

		// Trim @domain.name suffix if present
		username = strings.TrimSuffix(username, "@"+domain.NameOnly(service.host))

		// Trim path suffix if present
		username = list.First(username, '/')

	default:
		return digit.Resource{}, derp.New(derp.CodeBadRequestError, "service.User.LoadWebFinger", "Invalid username", username)
	}

	// Try to load the user from the database
	user := model.NewUser()
	if err := service.LoadByToken(username, &user); err != nil {
		return digit.Resource{}, derp.Wrap(err, "service.User.LoadWebFinger", "Error loading user", username)
	}

	// Make a WebFinger resource for this user.
	result := digit.NewResource("acct:"+username).
		Alias(user.GetProfileURL()).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, user.ActivityPubURL()).
		Link(digit.RelationTypeHub, model.MimeTypeJSONFeed, user.JSONFeedURL()).
		Link(digit.RelationTypeProfile, model.MimeTypeHTML, user.GetProfileURL()).
		Link(digit.RelationTypeAvatar, model.MimeTypeImage, user.ActivityPubAvatarURL()).
		Link(digit.RelationTypeSubscribeRequest, "", service.RemoteFollowURL())

	return result, nil
}

func (service *User) RemoteFollowURL() string {
	return service.host + "/.ostatus/tunnel?uri={uri}"
}
