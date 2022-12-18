package service

import (
	"context"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User manages all interactions with the User collection
type User struct {
	collection    data.Collection
	streamService *Stream
	emailService  *DomainEmail
	host          string
}

// NewUser returns a fully populated User service
func NewUser(collection data.Collection, streamService *Stream, emailService *DomainEmail, host string) User {
	service := User{
		streamService: streamService,
		emailService:  emailService,
		host:          host,
	}

	service.Refresh(collection)

	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *User) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *User) Close() {

}

/*******************************************
 * Common Data Methods
 *******************************************/

// List returns an iterator containing all of the Users who match the provided criteria
func (service *User) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an User from the database
func (service *User) Load(criteria exp.Expression, result *model.User) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.User", "Error loading User", criteria)
	}

	return nil
}

// Save adds/updates an User in the database
func (service *User) Save(user *model.User, note string) error {

	// RULE: Set ProfileURL to the hostname + the username
	user.ProfileURL = service.host + "/@" + user.Username

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
		return derp.Wrap(err, "service.User", "Error saving User", user, note)
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
		return derp.Wrap(err, "service.User", "Error deleting User", user, note)
	}

	// TODO: HIGH: Clean up related records (like InboxItem and OutboxItem)
	if err := service.streamService.DeleteByParent(user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, "service.User", "Error deleting User's streams", user, note)
	}

	return nil
}

/*******************************************
 * Generic Data Functions
 *******************************************/

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
	return service.collection.Query(result, criteria, options...)
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
	return derp.NewUnauthorizedError("service.User", "Not Authorized")
}

func (service *User) Schema() schema.Schema {
	return schema.New(model.UserSchema())
}

/*******************************************
 * Custom Queries
 *******************************************/

func (service *User) ListOwners() (data.Iterator, error) {
	return service.List(exp.Equal("isOwner", true))
}

func (service *User) ListOwnersAsSlice() []model.UserSummary {
	it, _ := service.ListOwners()
	return data.Slice(it, model.NewUserSummary)
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
		return derp.Wrap(err, "service.User", "Error loading User by ID", userID)
	}

	// If the password reset is not valid, then return an "Unauthorized" error
	if !user.PasswordReset.IsValid(code) {
		return derp.NewUnauthorizedError("service.User", "Invalid password reset code", userID, code)
	}

	// No Error means success
	return nil
}

// Count returns the number of (non-deleted) records in the User collection
func (service *User) Count(ctx context.Context, criteria exp.Expression) (int, error) {
	return queries.CountRecords(ctx, service.collection, notDeleted(criteria))
}

/*******************************************
 * Custom Actions
 *******************************************/

func (service *User) CalcFollowerCount(userID primitive.ObjectID) error {
	err := queries.SetFollowersCount(context.TODO(), service.collection, userID)
	return derp.Report(derp.Wrap(err, "service.User", "Error setting follower count", userID))
}

func (service *User) CalcFollowingCount(userID primitive.ObjectID) error {
	err := queries.SetFollowingCount(context.TODO(), service.collection, userID)
	return derp.Report(derp.Wrap(err, "service.User", "Error setting following count", userID))
}

func (service *User) CalcBlockCount(userID primitive.ObjectID) error {
	err := queries.SetBlockCount(context.TODO(), service.collection, userID)
	return derp.Report(derp.Wrap(err, "service.User", "Error setting block count", userID))
}

func (service *User) SetOwner(owner config.Owner) error {

	// Try to read the owner from the database
	users, err := service.ListOwners()

	if err != nil {
		return derp.Wrap(err, "service.User", "Error loading owners")
	}

	user := model.NewUser()
	found := false

	for users.Next(&user) {

		// If this user is already an owner, then we may be able to skip some work...
		if user.Username == owner.EmailAddress {
			found = true
		}

		// Update the user record and try to save it
		user.IsOwner = (user.Username == owner.EmailAddress)

		if err := service.Save(&user, "AssertOwner"); err != nil {
			return derp.Wrap(err, "service.User", "Error saving user", user)
		}
	}

	// If we didn't find an owner above, then we need to create one.
	if !found {
		user := model.NewUser()
		user.DisplayName = owner.DisplayName
		user.Username = owner.EmailAddress
		user.IsOwner = true

		if err := service.Save(&user, "CreateOwner"); err != nil {
			return derp.Wrap(err, "service.User", "Error saving user", user)
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

		return derp.Wrap(err, "service.User", "Error saving user", user)
	}

	return nil
}

// ResetPassword resets the password for the provided user
func (service *User) ResetPassword(user *model.User, resetCode string, newPassword string) error {

	// Verify that the password reset code is valid
	if !user.PasswordReset.IsValid(resetCode) {
		return derp.NewForbiddenError("service.User", "Invalid password reset code", user, resetCode)
	}

	// Update the password
	user.Password = newPassword

	// Invalidate the old reset code.
	user.PasswordReset = model.PasswordReset{}

	// Try to save the user with the new password reset code.
	if err := service.Save(user, "Create Password Reset Code"); err != nil {
		return derp.Wrap(err, "service.User", "Error saving user", user)
	}

	return nil
}

// SendWelcomeEmail generates a new password reset code and sends a welcome email to a new user.
// If there is a problem sending the email, then the new code is not saved.
func (service *User) SendWelcomeEmail(user *model.User) {

	if err := service.MakeNewPasswordResetCode(user); err != nil {
		derp.Report(derp.Wrap(err, "service.User", "Error making password reset", user))
		return
	}

	// Try to send the welcome email.  If it fails, then don't save the new password reset code.
	if err := service.emailService.SendWelcome(user); err != nil {
		derp.Report(derp.Wrap(err, "service.User", "Error sending welcome email", user))
		return
	}
}

// SendPasswordResetEmail generates a new password reset code and sends a welcome email to a new user.
// If there is a problem sending the email, then the new code is not saved.
func (service *User) SendPasswordResetEmail(user *model.User) {

	if err := service.MakeNewPasswordResetCode(user); err != nil {
		derp.Report(derp.Wrap(err, "service.User", "Error making password reset", user))
		return
	}

	// Try to send the welcome email.  If it fails, then don't save the new password reset code.
	if err := service.emailService.SendPasswordReset(user); err != nil {
		derp.Report(derp.Wrap(err, "service.User", "Error sending welcome email", user))
		return
	}
}

/*******************************************
 * WebFinger Behavior
 *******************************************/

func (service *User) LoadWebFinger(username string) (digit.Resource, error) {

	// Trim prefixes "acct:" and "@"
	username = strings.TrimPrefix(username, "acct:")
	username = strings.TrimPrefix(username, "@")

	// Trim @domain.name suffix if present
	username = strings.TrimSuffix(username, "@"+domain.NameOnly(service.host))

	// Try to load the user from the database
	user := model.NewUser()
	if err := service.LoadByUsername(username, &user); err != nil {
		return digit.Resource{}, derp.Wrap(err, "service.Stream.LoadWebFinger", "Error loading user", username)
	}

	// Make a WebFinger resource for this user.
	result := digit.NewResource("acct:"+username).
		Alias(user.ActivityPubProfileURL()).
		Link(digit.RelationTypeProfile, "text/html", user.ActivityPubProfileURL()).
		Link(digit.RelationTypeSelf, "application/activity+json", user.ActivityPubURL()).
		Link(digit.RelationTypeAvatar, "image/*", user.ActivityPubAvatarURL()).
		Link(digit.RelationTypeSubscribeRequest, "", user.ActivityPubSubscribeRequestURL())

	return result, nil
}
