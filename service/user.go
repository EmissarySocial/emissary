package service

import (
	"iter"
	"strconv"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/iterator"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User manages all interactions with the User collection
type User struct {
	collection        data.Collection
	followers         data.Collection
	following         data.Collection
	rules             data.Collection
	attachmentService *Attachment
	searchTagService  *SearchTag
	ruleService       *Rule
	emailService      *DomainEmail
	keyService        *EncryptionKey
	domainService     *Domain
	folderService     *Folder
	followerService   *Follower
	streamService     *Stream
	templateService   *Template
	webhookService    *Webhook
	activityStream    *ActivityStream
	queue             *queue.Queue
	host              string
}

// NewUser returns a fully populated User service
func NewUser() User {
	return User{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *User) Refresh(userCollection data.Collection, followerCollection data.Collection, followingCollection data.Collection, ruleCollection data.Collection, attachmentService *Attachment, domainService *Domain, emailService *DomainEmail, folderService *Folder, followerService *Follower, keyService *EncryptionKey, ruleService *Rule, searchTagService *SearchTag, streamService *Stream, templateService *Template, webhookService *Webhook, queue *queue.Queue, activityStream *ActivityStream, host string) {
	service.collection = userCollection
	service.searchTagService = searchTagService
	service.followers = followerCollection
	service.following = followingCollection
	service.rules = ruleCollection

	service.attachmentService = attachmentService
	service.domainService = domainService
	service.emailService = emailService
	service.folderService = folderService
	service.followerService = followerService
	service.keyService = keyService
	service.ruleService = ruleService
	service.streamService = streamService
	service.templateService = templateService
	service.webhookService = webhookService
	service.activityStream = activityStream
	service.queue = queue

	service.host = host
}

// Close stops any background processes controlled by this service
func (service *User) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// Count returns the number of Users who match the provided criteria
func (service User) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// List returns an iterator containing all of the Users who match the provided criteria
func (service *User) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns an iterator containing all of the Users who match the provided criteria
func (service *User) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.User], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.User.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewUser), nil
}

// Query returns an slice containing all of the Users who match the provided criteria
func (service *User) Query(criteria exp.Expression, options ...option.Option) ([]model.User, error) {
	result := make([]model.User, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
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

	const location = "service.User.Save"

	// RULE: DisplayName is required
	if user.EmailAddress == "" {
		return derp.NewBadRequestError(location, "EmailAddress is required", user)
	}

	// RULE: IF the display name is empty, then try the username and email address
	if user.DisplayName == "" {

		if user.Username != "" {
			user.DisplayName = user.Username
		} else if user.EmailAddress != "" {
			user.DisplayName = strings.Split(user.EmailAddress, "@")[0]
		} else {
			user.DisplayName = "New User"
		}
	}

	// RULE: If the username is empty, then try to automatically generate one
	if err := service.CalcUsername(user); err != nil {
		return derp.Wrap(err, location, "Error calculating username", user)
	}

	isNew := user.IsNew()

	// Special steps to take on initial creation
	if isNew {

		// RULE: Set DEFAULT inbox/outbox values based on the Theme
		theme := service.domainService.Theme()

		if user.InboxTemplate == "" {
			user.InboxTemplate = theme.DefaultInbox
		}

		if user.OutboxTemplate == "" {
			user.OutboxTemplate = theme.DefaultOutbox
		}
	}

	// RULE: Set ProfileURL to the hostname + the username
	user.ProfileURL = service.host + "/@" + user.UserID.Hex()

	// RULE: If password reset has already expired, then clear the reset code
	if (user.PasswordReset.ExpireDate > 0) && (user.PasswordReset.ExpireDate < time.Now().Unix()) {
		user.PasswordReset.AuthCode = ""
	}

	// Validate the value before saving
	if err := service.Schema().Validate(user); err != nil {
		return derp.Wrap(err, location, "Invalid User Data", user)
	}

	// Try to save the User record to the database
	if err := service.collection.Save(user, note); err != nil {
		return derp.Wrap(err, location, "Error saving User", user, note)
	}

	// RULE: Take these actions when setting up a new user
	if isNew {

		// RULE: Create a new encryption key for this user
		if _, err := service.keyService.Create(model.EncryptionKeyTypeUser, user.UserID); err != nil {
			return derp.Wrap(err, location, "Error creating encryption key for User", user, note)
		}

		// RULE: Create default folders for this user
		if err := service.folderService.CreateDefaultFolders(user.UserID); err != nil {
			return derp.Wrap(err, location, "Error creating default folders for User", user, note)
		}
	}

	// Send Webhooks (if configured)
	if isNew {
		service.webhookService.Send(user, model.WebhookEventUserCreate)
	} else {
		service.webhookService.Send(user, model.WebhookEventUserUpdate)
	}

	// Success!
	return nil
}

// Delete removes an User from the database (virtual delete)
func (service *User) Delete(user *model.User, note string) error {

	// TODO: Delete related folders

	// TODO: Delete related followers

	// TODO: Delete related following

	// TODO: Delete related group memberships

	// TODO: Delete related mentions

	// TODO: Delete related inbox messages

	// TODO: Delete related outbox messages

	// TODO: Delete related rules

	// Delete related streams
	if err := service.streamService.DeleteByParent(user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, "service.User.Delete", "Error deleting User's streams", user, note)
	}

	// TODO: Delete related tags

	// Delete the User from the database
	if err := service.collection.Delete(user, note); err != nil {
		return derp.Wrap(err, "service.User.Delete", "Error deleting User", user, note)
	}

	// Send user:delete webhooks
	service.webhookService.Send(user, model.WebhookEventUserDelete)

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

func (service *User) RangeAll() (iter.Seq[model.User], error) {
	return service.Range(exp.All())
}

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

// LoadByMapID loads a single model.User object that matches the provided mapID key/value
func (service *User) LoadByMapID(key string, value string, result *model.User) error {
	criteria := exp.Equal("mapIds."+key, value)
	return service.Load(criteria, result)
}

// LoadByProfileURL loads a single model.User object that matches the provided profile URL
func (service *User) LoadByProfileURL(profileUrl string, result *model.User) error {
	criteria := exp.Equal("profileUrl", profileUrl)
	return service.Load(criteria, result)
}

// LoadByUsername loads a single model.User object that matches the provided username
func (service *User) LoadByUsername(username string, result *model.User) error {
	criteria := exp.Equal("username", username)
	return service.Load(criteria, result)
}

// LoadByUsernameOrEmail loads a single model.User object that matches the provided username or email address
func (service *User) LoadByUsernameOrEmail(usernameOrEmail string, result *model.User) error {
	criteria := exp.Equal("username", usernameOrEmail).OrEqual("emailAddress", usernameOrEmail)
	err := service.Load(criteria, result)

	return err
}

// LoadByEmail loads a single model.User object that matches the provided email address
func (service *User) LoadByEmail(email string, result *model.User) error {
	criteria := exp.Equal("emailAddress", email)
	err := service.Load(criteria, result)

	return err
}

// LoadByUsername loads a single model.User object that matches the provided token.
// If the "token" is a valid ObjectID, then it attempts to load by that userID.
// If the "token" is not a valid ObjectID (or if the first attempt fails), then it tries to load by username.
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

	const location = "service.User.LoadByResetCode"

	// Try to find the user by ID
	if err := service.LoadByToken(userID, user); err != nil {
		return derp.Wrap(err, location, "Error loading User by ID", userID)
	}

	// If the password reset is not valid, then return an "Unauthorized" error
	if !user.PasswordReset.IsValid(code) {
		return derp.NewUnauthorizedError(location, "Invalid password reset code", userID, code)
	}

	// No Error means success
	return nil
}

// TODO: MEDIUM: this function is wickedly inefficient
// Should probably use a RuleFilter here.
func (service *User) QueryBlockedActors(userID primitive.ObjectID, criteria exp.Expression) ([]model.User, error) {

	const location = "service.User.QueryBlockedUsers"

	// Query all rules
	rules, err := service.ruleService.QueryBlockedActors(userID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error querying rules")
	}

	// Extract the blocked userIDs
	blockedUserIDs := slice.Map(rules, func(rule model.Rule) string {
		return rule.Trigger
	})

	// Query all users
	return service.Query(criteria.AndEqual("_id", blockedUserIDs), option.SortAsc("createDate"))
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *User) CalcUsername(user *model.User) error {

	// If the User has a valid username, then there's nothing to do.
	if user.Username != "" {
		return nil
	}

	// Calculate the new base username
	base := first.String(user.DisplayName, user.EmailAddress, user.UserID.Hex())
	base = strings.ToLower(base)
	base = strings.ReplaceAll(base, " ", "")
	base = strings.ReplaceAll(base, ".", "")
	base, _, _ = strings.Cut(base, "@")

	// Try to use the preferred username with no slug
	if !service.usernameExists(user.UserID, base) {
		user.Username = base
		return nil
	}

	// Otherwise, try slug values until we find a unique username (max 32)
	for i := 1; i < 32; i++ {
		slug := random.GenerateInt(1000, 9999)
		username := base + strconv.Itoa(slug)

		if !service.usernameExists(user.UserID, username) {
			user.Username = username
			return nil
		}
	}

	// Okay, this sucks, but we need to call it here.  Return error.
	return derp.NewInternalError("service.User.CalcUsername", "Unable to generate a unique username", user)
}

// usernameExists returns TRUE if the provided username is already in use by another user
func (service *User) usernameExists(userID primitive.ObjectID, username string) bool {
	user := model.NewUser()
	criteria := exp.Equal("username", username).AndNotEqual("_id", userID)

	// Try to find a User with the same username and a different ID
	err := service.Load(criteria, &user)

	// If found, return TRUE.  If NOT found, return FALSE.
	return err == nil
}

func (service *User) CalcFollowerCount(userID primitive.ObjectID) {
	if err := queries.SetFollowersCount(service.collection, service.followers, userID); err != nil {
		derp.Report(derp.Wrap(err, "service.User.CalcFollowerCount", "Error setting follower count", userID))
	}
}

func (service *User) CalcFollowingCount(userID primitive.ObjectID) {
	if err := queries.SetFollowingCount(service.collection, service.following, userID); err != nil {
		derp.Report(derp.Wrap(err, "service.User.CalcFollowingCount", "Error setting following count", userID))
	}
}

func (service *User) CalcRuleCount(userID primitive.ObjectID) {
	if err := queries.SetRuleCount(service.collection, service.rules, userID); err != nil {
		derp.Report(derp.Wrap(err, "service.User.CalcRuleCount", "Error setting rule count", userID))
	}
}

func (service *User) SetOwner(owner config.Owner) error {

	// If there is no owner data, then do not create/update an owner record.
	if owner.IsEmpty() {
		return nil
	}

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

func (service *User) DeleteAvatar(user *model.User, note string) error {

	// If there is no image, then there's nothing more to do.
	if user.IconID.IsZero() {
		return nil
	}

	// Delete the existing Avatar file
	if err := service.attachmentService.DeleteByID(model.AttachmentObjectTypeUser, user.UserID, user.IconID, note); err != nil {
		return derp.Wrap(err, "service.User.DeleteAvatar", "Error deleting avatar", user)
	}

	// Clear the reference in the User object
	user.IconID = primitive.NilObjectID
	if err := service.Save(user, note); err != nil {
		return derp.Wrap(err, "service.User.DeleteAvatar", "Error saving user", user)
	}

	return nil
}

/******************************************
 * Email Methods
 ******************************************/

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

// MakeNewPasswordResetCode generates a new password reset code for the provided user.
func (service *User) MakeNewPasswordResetCode(user *model.User) error {

	// If the PasswordReset IS NOT active then
	// create a new password reset code for this user
	if user.PasswordReset.NotActive() {
		user.PasswordReset = model.NewPasswordReset()
	}

	// In all cases, refresh the expiration date of the password reset code
	// so that it can be used for another 24 hours.
	user.PasswordReset.RefreshExpireDate()

	// Try to save the user with the new password reset code.
	if err := service.Save(user, "Create Password Reset Code"); err != nil {
		return derp.Wrap(err, "service.User.MakeNewPasswordResetCode", "Error saving user", user)
	}

	return nil
}

/******************************************
 * WebFinger Behavior
 ******************************************/

func (service *User) LoadWebFinger(username string) (digit.Resource, error) {

	const location = "service.User.LoadWebFinger"

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
		return digit.Resource{}, derp.NewBadRequestError(location, "Invalid username", username)
	}

	// Try to load the user from the database
	user := model.NewUser()
	if err := service.LoadByToken(username, &user); err != nil {
		return digit.Resource{}, derp.Wrap(err, location, "Error loading user", username)
	}

	// Make a WebFinger resource for this user.
	result := digit.NewResource("acct:"+username+"@"+domain.NameOnly(service.host)).
		Alias(service.host+"/@"+user.Username).
		Alias(user.ActivityPubURL()).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, user.ActivityPubURL()).
		Link(digit.RelationTypeHub, model.MimeTypeJSONFeed, user.JSONFeedURL()).
		Link(digit.RelationTypeProfile, model.MimeTypeHTML, user.ActivityPubURL()).
		Link(digit.RelationTypeAvatar, model.MimeTypeImage, user.ActivityPubIconURL()).
		Link(digit.RelationTypeSubscribeRequest, "", service.RemoteFollowURL()).
		Link(camper.IntentTypeCreate, "", service.CreateIntentURL()).
		Link(camper.IntentTypeDislike, "", service.DislikeIntentURL()).
		Link(camper.IntentTypeFollow, "", service.FollowIntentURL()).
		Link(camper.IntentTypeLike, "", service.LikeIntentURL())

	return result, nil
}

func (service *User) RemoteFollowURL() string {
	return service.host + "/.ostatus/tunnel?uri={uri}"
}

func (service *User) CreateIntentURL() string {
	return service.host + "/@me/intent/create?type={type}&name={name}&summary={summary}&content={content}&inReplyTo={inReplyTo}&on-success={on-success}&on-cancel={on-cancel}"
}

func (service *User) DislikeIntentURL() string {
	return service.host + "/@me/intent/dislike?object={object}&on-success={on-success}&on-cancel={on-cancel}"
}

func (service *User) FollowIntentURL() string {
	return service.host + "/@me/intent/follow?object={object}&on-success={on-success}&on-cancel={on-cancel}"
}

func (service *User) LikeIntentURL() string {
	return service.host + "/@me/intent/like?object={object}&on-success={on-success}&on-cancel={on-cancel}"
}

func (service *User) CalculateTags(user *model.User) {

	const location = "service.User.CalculateTags"

	// Load the Template (to get TagPaths)
	template, err := service.templateService.Load(user.OutboxTemplate)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error loading template", user.OutboxTemplate))
		return
	}

	// Prepare to scan all TagPaths for #hashtags
	schema := service.Schema()
	hashtags := sliceof.NewString()

	for _, path := range template.TagPaths {

		if value, err := schema.Get(user, path); err == nil {

			// Massage the value into a cleanly searchable string
			stringValue := convert.String(value)
			stringValue = html.ToSearchText(stringValue)
			hashtags = append(hashtags, parse.Hashtags(stringValue)...)
		}
	}

	// Look up normalized hashtag names in the database
	hashtagNames, _, err := service.searchTagService.NormalizeTags(hashtags...)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error normalizing tags", hashtags))
	}

	// Apply the normalized hashtag names to the user object
	user.Hashtags = hashtagNames
}

/******************************************
 * SearchResulter Interface
 ******************************************/

func (service *User) SearchResult(user *model.User) (model.SearchResult, bool) {

	if !user.IsIndexable {
		return model.SearchResult{}, false
	}

	result := model.NewSearchResult()

	result.Type = "Person"
	result.Name = user.DisplayName
	result.AttributedTo = "@" + user.Username
	result.Summary = user.StatusMessage
	result.URL = user.ProfileURL
	result.IconURL = user.ActivityPubIconURL()
	result.TagNames = user.Hashtags
	result.TagValues = slice.Map(user.Hashtags, model.ToToken)
	result.FullText = user.DisplayName + ", " + user.Username + ", " + user.Location + ", " + user.StatusMessage

	return result, true
}
