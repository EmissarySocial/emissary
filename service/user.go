package service

import (
	"iter"
	"strconv"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/iterator"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/schema/format"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/steranko"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User manages all interactions with the User collection
type User struct {
	activityService   *ActivityStream
	attachmentService *Attachment
	connectionService *Connection
	emailService      *DomainEmail
	domainService     *Domain
	folderService     *Folder
	followerService   *Follower
	followingService  *Following
	keyService        *EncryptionKey
	newsFeedService   *NewsFeed
	outboxService     *Outbox
	outbox2Service    *Outbox2
	responseService   *Response
	ruleService       *Rule
	searchTagService  *SearchTag
	steranko          func(data.Session) *steranko.Steranko
	streamService     *Stream
	templateService   *Template
	webhookService    *Webhook
	queue             *queue.Queue
	sseUpdateChannel  chan<- realtime.Message
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
func (service *User) Refresh(factory *Factory) {

	service.activityService = factory.ActivityStream()
	service.attachmentService = factory.Attachment()
	service.connectionService = factory.Connection()
	service.domainService = factory.Domain()
	service.emailService = factory.Email()
	service.folderService = factory.Folder()
	service.followerService = factory.Follower()
	service.followingService = factory.Following()
	service.newsFeedService = factory.NewsFeed()
	service.keyService = factory.EncryptionKey()
	service.outboxService = factory.Outbox()
	service.outbox2Service = factory.Outbox2()
	service.responseService = factory.Response()
	service.ruleService = factory.Rule()
	service.steranko = factory.Steranko
	service.streamService = factory.Stream()
	service.templateService = factory.Template()
	service.webhookService = factory.Webhook()
	service.sseUpdateChannel = factory.SSEUpdateChannel()
	service.queue = factory.Queue()

	service.host = factory.Host()
}

// Hostname returns the domain-only name (no protocol)
func (service *User) Hostname() string {
	return uri.Hostname(service.host)
}

// Host returns the host (with protocol)
func (service *User) Host() string {
	return service.host
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the mongo collection where Users are stored
func (service *User) collection(session data.Session) data.Collection {
	return session.Collection("User")
}

// followerCollection returns the mongo collection where Followers are stored
func (service *User) followerCollection(session data.Session) data.Collection {
	return session.Collection("Follower")
}

// followingCollection returns the mongo collection where Following records are stored
func (service *User) followingCollection(session data.Session) data.Collection {
	return session.Collection("Following")
}

// ruleCollection returns the mongo collection where Rules are stored
func (service *User) ruleCollection(session data.Session) data.Collection {
	return session.Collection("Rule")
}

// Count returns the number of Users who match the provided criteria
func (service User) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// List returns an iterator containing all of the Users who match the provided criteria
func (service *User) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns an iterator containing all of the Users who match the provided criteria
func (service *User) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.User], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.User.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewUser), nil
}

// Query returns an slice containing all of the Users who match the provided criteria
func (service *User) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.User, error) {
	result := make([]model.User, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Load retrieves an User from the database
func (service *User) Load(session data.Session, criteria exp.Expression, result *model.User, options ...option.Option) error {
	if err := service.collection(session).Load(notDeleted(criteria), result, options...); err != nil {
		return derp.Wrap(err, "service.User.Load", "Loading User", criteria)
	}

	return nil
}

// Save adds/updates an User in the database
func (service *User) Save(session data.Session, user *model.User, note string) error {

	const location = "service.User.Save"

	// RULE: EmailAddress is required
	if user.EmailAddress == "" {
		return derp.BadRequest(location, "EmailAddress is required", user)
	}

	// RULE: IF the display name is empty, then try the username and email address
	if user.DisplayName == "" {

		if user.Username != "" {
			user.DisplayName = user.Username
		} else if user.EmailAddress != "" {
			user.DisplayName, _, _ = strings.Cut(user.EmailAddress, "@")
		} else {
			user.DisplayName = "New User"
		}
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

		// RULE: If the username is empty, then try to automatically generate one
		if err := service.CalcNewUsername(session, user); err != nil {
			return derp.Wrap(err, location, "Calculating username", user)
		}
	}

	// RULE: Denormalize the tag URL from the outbox Template and extract #hashtags from profile
	// fields.  Linkification itself happens at render time (User.SummaryHTML), not here, because
	// statusMessage stores raw Markdown that is rendered to HTML at display time.
	if template, err := service.templateService.Load(user.OutboxTemplate); err == nil {
		user.TagURL = template.TagURL

		if len(template.TagPaths) > 0 {
			service.CalculateTags(session, user)
		}
	}

	// Normalize the value before saving.  Values are rewritten in place (formatted,
	// clamped, truncated) to conform to the schema, so that legacy data written under
	// older rules is repaired progressively as records are saved.
	//
	// This runs BEFORE ValidateUsername so that uniqueness and formatting rules are
	// checked against the username as it will actually be stored (e.g. after truncation
	// to the schema's max length), not the raw client-supplied value.  Otherwise an
	// over-length username could pass uniqueness against its full form, then be truncated
	// into a collision with an existing account.
	rewrites, err := service.Schema().Normalize(user)

	if err != nil {
		return derp.Wrap(err, location, "Invalid User Data", user)
	}

	if len(rewrites) > 0 {
		log.Debug().Strs("rewrites", rewrites).Str("userId", user.UserID.Hex()).Msg("User values normalized during save")
	}

	// Guarantee that the (normalized) username is unique, and fits formatting rules.
	if err := service.ValidateUsername(session, user.UserID, user.Username); err != nil {
		return derp.Wrap(err, location, "Username is invalid", user)
	}

	// RULE: Set ProfileURL to the hostname + the username
	user.ProfileURL = service.host + "/@" + user.UserID.Hex()

	// RULE: If password reset has already expired, then clear the reset code
	if (user.PasswordReset.ExpireDate > 0) && (user.PasswordReset.ExpireDate < time.Now().Unix()) {
		user.PasswordReset.AuthCode = ""
	}

	// Fingerprint the public actor document to detect profile changes worth federating.
	// An empty stored fingerprint (pre-feature record) counts as changed: better one chatty
	// Update per user during the transition than a missed one (PROFILE-UPDATE-FEDERATION.md D-6).
	newFingerprint, err := user.CalcProfileFingerprint()

	if err != nil {
		return derp.Wrap(err, location, "Calculating profile fingerprint", user)
	}

	profileChanged := (user.ProfileFingerprint != newFingerprint) && !isNew
	user.ProfileFingerprint = newFingerprint

	// Try to save the User record to the database
	if err := service.collection(session).Save(user, note); err != nil {
		return derp.Wrap(err, location, "Saving User", user, note)
	}

	// RULE: Take these actions when setting up a new user
	if isNew {

		// RULE: Create a new encryption key for this user
		if _, err := service.keyService.Create(session, model.EncryptionKeyTypeUser, user.UserID); err != nil {
			return derp.Wrap(err, location, "Creating encryption key for User", user, note)
		}

		// RULE: Create default folders for this user
		if err := service.folderService.CreateDefaultFolders(session, user.UserID); err != nil {
			return derp.Wrap(err, location, "Creating default folders for User", user, note)
		}
	}

	// Handle Bluesky bridging opt-in/out
	if err := service.connectBluesky(session, user); err != nil {
		return derp.Wrap(err, location, "Updating bridge settings", user, note)
	}

	// Set AttributedTo value
	service.streamService.SetAttributedTo(user)

	// RULE: Federate profile changes to followers via an ActivityPub Update (post-commit)
	if profileChanged {
		if err := service.sendProfileUpdate(session, user); err != nil {
			return derp.Wrap(err, location, "Sending profile Update", user)
		}
	}

	// Send Webhooks (if configured)
	eventName := iif(isNew, model.WebhookEventUserCreate, model.WebhookEventUserUpdate)
	service.webhookService.Send(user, eventName)

	// Notify SSE clients that this User has been updated
	service.sseUpdateChannel <- realtime.NewMessage_Updated(user.UserID)

	// Success!
	return nil
}

// Delete removes a User from the database (virtual delete), along with all of their related records
func (service *User) Delete(session data.Session, user *model.User, note string) error {

	const location = "service.User.Delete"

	// Delete related Folders
	if err := service.folderService.DeleteByUserID(session, user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, location, "Deleting User's folders", user, note)
	}

	// Delete related Followers
	if err := service.followerService.DeleteByUserID(session, user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, location, "Deleting User's followers", user, note)
	}

	// Delete related Following
	if err := service.followingService.DeleteByUserID(session, user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, location, "Deleting User's following", user, note)
	}

	// TODO: Delete related mentions

	// Delete related Encryption Keys messages
	if err := service.keyService.DeleteByParentID(session, user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, location, "Deleting User's encryption keys", user, note)
	}

	// Delete related NewsFeed messages
	if err := service.newsFeedService.DeleteByUserID(session, user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, location, "Deleting User's inbox messages", user, note)
	}

	// Delete related Outbox messages
	if err := service.outboxService.DeleteByParentID(session, model.FollowerTypeUser, user.UserID); err != nil {
		return derp.Wrap(err, location, "Deleting User's outbox messages", user, note)
	}

	// Delete related Responses
	if err := service.responseService.DeleteByUserID(session, user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, location, "Deleting User's responses", user, note)
	}

	// Delete related Rules
	if err := service.ruleService.DeleteByUserID(session, user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, location, "Deleting User's rules", user, note)
	}

	// Delete related Streams
	if err := service.streamService.DeleteByParent(session, user.UserID, "Deleted with owner"); err != nil {
		return derp.Wrap(err, location, "Deleting User's streams", user, note)
	}

	// Delete the User from the database
	if err := service.collection(session).Delete(user, note); err != nil {
		return derp.Wrap(err, location, "Deleting User", user, note)
	}

	// Send user:delete webhooks
	service.webhookService.Send(user, model.WebhookEventUserDelete)

	// Farewell, sweet prince
	return nil
}

/******************************************
 * Generic Data Functions
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *User) ObjectType() string {
	return "User"
}

// ObjectNew returns a fully initialized model.User as a data.Object.
func (service *User) ObjectNew() data.Object {
	result := model.NewUser()
	return &result
}

// ObjectID returns the primary key of the provided User object
func (service *User) ObjectID(object data.Object) primitive.ObjectID {

	if user, ok := object.(*model.User); ok {
		return user.UserID
	}

	return primitive.NilObjectID
}

// ObjectQuery populates the result value with all Users who match the provided criteria
func (service *User) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single User who matches the provided criteria, as a data.Object
func (service *User) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewUser()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave saves the provided User object to the database
func (service *User) ObjectSave(session data.Session, object data.Object, note string) error {
	if user, ok := object.(*model.User); ok {
		return service.Save(session, user, note)
	}
	return derp.Internal("service.User.ObjectSave", "Invalid object type", object)
}

// ObjectDelete removes the provided User object from the database (virtual delete)
func (service *User) ObjectDelete(session data.Session, object data.Object, note string) error {
	if user, ok := object.(*model.User); ok {
		return service.Delete(session, user, note)
	}
	return derp.Internal("service.User.ObjectDelete", "Invalid object type", object)
}

// ObjectUserCan always returns Unauthorized: Users are never edited through the generic data.Object path
func (service *User) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.User.ObjectUserCan", "Not Authorized")
}

// Schema returns the validating schema for User objects
func (service *User) Schema() schema.Schema {
	return schema.New(model.UserSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// RangeAll returns an iterator containing every User in the database
func (service *User) RangeAll(session data.Session) (iter.Seq[model.User], error) {
	return service.Range(session, exp.All())
}

// ListUsernameOrOwner returns an iterator of all Users who are owners OR match the provided username
func (service *User) ListUsernameOrOwner(session data.Session, username string) (data.Iterator, error) {
	return service.List(session, exp.Equal("isOwner", true).OrEqual("username", username))
}

// ListOwners returns an iterator of all Users who are marked as owners
func (service *User) ListOwners(session data.Session) (data.Iterator, error) {
	return service.List(session, exp.Equal("isOwner", true))
}

// ListOwnersAsSlice returns a slice of UserSummaries for all Users who are marked as owners
func (service *User) ListOwnersAsSlice(session data.Session) []model.UserSummary {
	it, _ := service.ListOwners(session)
	return iterator.Slice(it, model.NewUserSummary)
}

// ListByIdentities returns all users that appear in the list of identities
func (service *User) ListByIdentities(session data.Session, identities []string) (data.Iterator, error) {
	return service.List(session, exp.In("identities", identities))
}

// RangeByGroup returns an iterator of all Users that match a provided groupID
func (service *User) RangeByGroup(session data.Session, group primitive.ObjectID) (iter.Seq[model.User], error) {
	return service.Range(session, exp.Equal("groupIds", group))
}

// LoadByID loads a single model.User object that matches the provided userID
func (service *User) LoadByID(session data.Session, userID primitive.ObjectID, result *model.User) error {
	criteria := exp.Equal("_id", userID)
	return service.Load(session, criteria, result)
}

// LoadByMapID loads a single model.User object that matches the provided mapID key/value
func (service *User) LoadByMapID(session data.Session, key string, value string, result *model.User) error {
	criteria := exp.Equal("mapIds."+key, value)
	return service.Load(session, criteria, result)
}

// LoadByProfileURL loads a single model.User object that matches the provided profile URL
func (service *User) LoadByProfileURL(session data.Session, profileUrl string, result *model.User) error {
	criteria := exp.Equal("profileUrl", profileUrl)
	return service.Load(session, criteria, result, option.CaseSensitive(false))
}

// LoadByUsername loads a single model.User object that matches the provided username
func (service *User) LoadByUsername(session data.Session, username string, result *model.User) error {
	criteria := exp.Equal("username", username)
	return service.Load(session, criteria, result, option.CaseSensitive(false))
}

// LoadByUsernameOrEmail loads a single model.User object that matches the provided username or email address
func (service *User) LoadByUsernameOrEmail(session data.Session, usernameOrEmail string, result *model.User) error {
	criteria := exp.Equal("username", usernameOrEmail).OrEqual("emailAddress", usernameOrEmail)
	err := service.Load(session, criteria, result, option.CaseSensitive(false))

	return err
}

// LoadByEmail loads a single model.User object that matches the provided email address
func (service *User) LoadByEmail(session data.Session, email string, result *model.User) error {
	criteria := exp.Equal("emailAddress", email)
	err := service.Load(session, criteria, result, option.CaseSensitive(false))

	return err
}

// LoadByToken loads a single model.User object that matches the provided token.
// If the "token" is a valid ObjectID, then it attempts to load by that userID.
// If the "token" is not a valid ObjectID (or if the first attempt fails), then it tries to load by username.
func (service *User) LoadByToken(session data.Session, token string, result *model.User) error {

	// If the token is an ObjectID then try that first.
	if userID, err := primitive.ObjectIDFromHex(token); err == nil {
		return service.LoadByID(session, userID, result)
	}

	// Otherwise, use the token as a username
	return service.LoadByUsername(session, token, result)
}

// LoadByResetCode loads a single model.User, but only when the provided password reset code is valid
func (service *User) LoadByResetCode(session data.Session, userID string, code string, user *model.User) error {

	const location = "service.User.LoadByResetCode"

	// Try to find the user by ID
	if err := service.LoadByToken(session, userID, user); err != nil {
		return derp.Wrap(err, location, "Loading User by ID", userID)
	}

	// If the password reset is not valid, then return an "Unauthorized" error
	if !user.PasswordReset.IsValid(code) {
		return derp.Unauthorized(location, "Invalid password reset code", userID, code)
	}

	// No Error means success
	return nil
}

// QueryBlockedActors returns all local Users that the provided User has blocked.
// TODO: MEDIUM: this function is wickedly inefficient; should probably use a RuleFilter here.
func (service *User) QueryBlockedActors(session data.Session, userID primitive.ObjectID, criteria exp.Expression) ([]model.User, error) {

	const location = "service.User.QueryBlockedActors"

	// Query all rules
	rules, err := service.ruleService.QueryBlockedActors(session, userID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Querying rules")
	}

	// Extract the blocked userIDs
	blockedUserIDs := slice.Map(rules, func(rule model.Rule) string {
		return rule.Trigger
	})

	// Query all users
	return service.Query(session, criteria.AndEqual("_id", blockedUserIDs), option.SortAsc("createDate"))
}

/******************************************
 * Custom Actions
 ******************************************/

// Shuffle assigns a unique random number to the "shuffle" field of each User
func (service *User) Shuffle(session data.Session) error {

	collection := service.collection(session)
	if err := queries.Shuffle(session.Context(), collection); err != nil {
		return derp.Wrap(err, "service.User.Shuffle", "Shuffling users")
	}

	return nil
}

// CalcNewUsername generates a unique username for a new User, derived from their
// display name or email address, appending a random slug on collisions.
func (service *User) CalcNewUsername(session data.Session, user *model.User) error {

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
	if !service.UsernameExists(session, user.UserID, base) {
		user.Username = base
		return nil
	}

	// Otherwise, try slug values until we find a unique username (max 32)
	for i := 1; i < 32; i++ {
		slug := random.GenerateInt(1000, 9999)

		if username := base + strconv.Itoa(slug); !service.UsernameExists(session, user.UserID, username) {
			user.Username = username
			return nil
		}
	}

	// Okay, this sucks, but we need to call it here.  Return error.
	return derp.Internal("service.User.CalcUsername", "Unable to generate a unique username", user)
}

// ValidateUsername returns an error unless the provided username is present, unreserved,
// well-formatted, and unique among all other Users.
func (service *User) ValidateUsername(session data.Session, userID primitive.ObjectID, username string) error {

	const location = "service.User.ValidateUsername"

	switch username {

	// RULE: Username is required
	case "":
		return derp.BadRequest(location, "Username is required", username)

	// RULE: Reserved names cannot be used
	case
		"admin",
		"administrator",
		"application",
		"guest",
		"identity",
		"me",
		"owner",
		"root",
		"search",
		"service",
		"system",
		"test",
		"user":

		return derp.BadRequest(location, "Username is not allowed", username)
	}

	// RULE: Username must be at least 3 characters
	if len(username) < 3 {
		return derp.BadRequest(location, "Username must be at least 3 characters", username)
	}

	// RULE: Username can only contain letters, numbers, and underscores
	if _, err := format.Username("")(username); err != nil {
		return derp.Wrap(err, location, "Username must contain only: letters, numbers, and underscores.", username)
	}

	// RULE: Username must be unique
	if service.UsernameExists(session, userID, username) {
		return derp.BadRequest(location, "Username is already in use", username)
	}

	return nil
}

// UsernameExists returns TRUE if the provided username is already in use by another user
func (service *User) UsernameExists(session data.Session, userID primitive.ObjectID, username string) bool {
	user := model.NewUser()

	criteria := exp.Equal("username", username).
		AndNotEqual("_id", userID)

	// Try to find a User with the same username and a different ID
	err := service.Load(session, criteria, &user)

	// If found, return TRUE.  If NOT found, return FALSE.
	return err == nil
}

// CalcFollowerCount recalculates the denormalized follower count for the provided User
func (service *User) CalcFollowerCount(session data.Session, userID primitive.ObjectID) error {

	const location = "service.User.CalcFollowerCount"

	// RULE: If UserID is zero, this is a domain actor that doesn't need to be calc'ed
	if userID.IsZero() {
		return nil
	}

	userCollection := service.collection(session)
	followersCollection := service.followerCollection(session)
	if err := queries.SetFollowersCount(userCollection, followersCollection, userID); err != nil {
		return derp.Wrap(err, location, "Counting `Follower` records", userID)
	}

	return nil
}

// CalcFollowingCount recalculates the denormalized following count for the provided User
func (service *User) CalcFollowingCount(session data.Session, userID primitive.ObjectID) error {

	const location = "service.User.CalcFollowingCount"

	userCollection := service.collection(session)
	followingCollection := service.followingCollection(session)

	if err := queries.SetFollowingCount(userCollection, followingCollection, userID); err != nil {
		return derp.Wrap(err, location, "Counting `Following` records", userID)
	}

	return nil
}

// CalcRuleCount recalculates the denormalized rule count for the provided User
func (service *User) CalcRuleCount(session data.Session, userID primitive.ObjectID) error {

	const location = "service.User.CalcRuleCount"

	// RULE: We only need to count rules owned by a specific user.
	if userID.IsZero() {
		return nil
	}

	userCollection := service.collection(session)
	rulesCollection := service.ruleCollection(session)

	if err := queries.SetRuleCount(userCollection, rulesCollection, userID); err != nil {
		return derp.Wrap(err, location, "Counting rules", userID)
	}

	return nil
}

// SetOwner creates or updates the User record matching the owner in the domain
// configuration, and clears the owner flag from every other User.
func (service *User) SetOwner(session data.Session, owner config.Owner) error {

	const location = "service.User.SetOwner"

	// If there is no owner data, then do not create/update an owner record.
	if owner.IsEmpty() {
		return nil
	}

	// Try to read the owner from the database
	users, err := service.ListUsernameOrOwner(session, owner.Username)

	if err != nil {
		return derp.Wrap(err, location, "Loading owners")
	}

	found := false

	for user := model.NewUser(); users.Next(&user); user = model.NewUser() {

		// See if this user is the "owner" being added/updated
		isOwner := (user.Username == owner.Username)

		// Mark "found" if possible
		if isOwner {
			found = true
		}

		// If we're changing this record, then save it.
		if user.IsOwner != isOwner {
			user.IsOwner = isOwner

			if err := service.Save(session, &user, "Set Owner"); err != nil {
				return derp.Wrap(err, location, "Saving user", user)
			}
		}
	}

	// If we didn't find an owner above, then we need to create one.
	if !found {
		user := model.NewUser()
		user.DisplayName = owner.DisplayName
		user.EmailAddress = owner.EmailAddress
		user.Username = owner.Username
		user.IsOwner = true

		if err := service.Save(session, &user, "CreateOwner"); err != nil {
			return derp.Wrap(err, location, "Saving user", user)
		}
	}

	return nil
}

// DeleteAvatar removes the User's avatar attachment and clears its reference on the User record
func (service *User) DeleteAvatar(session data.Session, user *model.User, note string) error {

	const location = "service.User.DeleteAvatar"

	// If there is no image, then there's nothing more to do.
	if user.IconID.IsZero() {
		return nil
	}

	// Delete the existing Avatar file
	if err := service.attachmentService.DeleteByID(session, model.AttachmentObjectTypeUser, user.UserID, user.IconID, note); err != nil {
		return derp.Wrap(err, location, "Deleting avatar attachment", user)
	}

	// Clear the reference in the User object
	user.IconID = primitive.NilObjectID
	if err := service.Save(session, user, note); err != nil {
		return derp.Wrap(err, location, "Saving User", user)
	}

	return nil
}

/******************************************
 * Email Methods
 ******************************************/

// SendPasswordResetEmail generates a new password reset code (valid for the provided duration) and
// emails it to the user.  The error is RETURNED (not swallowed) so callers on member-facing flows can
// tell the member the email could not be sent, instead of pointing them at an inbox that will never
// receive it.  NOTE: the reset code is persisted by MakeNewPasswordResetCode BEFORE the email is sent,
// and a send failure does NOT roll it back -- so a code issued here stays valid for a later retry.
func (service *User) SendPasswordResetEmail(session data.Session, user *model.User, duration time.Duration) error {

	const location = "service.User.SendPasswordResetEmail"

	if err := service.MakeNewPasswordResetCode(session, user, duration); err != nil {
		return derp.Wrap(err, location, "Making password reset", user)
	}

	if err := service.emailService.SendPasswordReset(user); err != nil {
		return derp.Wrap(err, location, "Sending password reset", user)
	}

	return nil
}

// NotifySigninLockout emails the account owner that their account has been
// temporarily locked after repeated failed signin attempts. This method swallows
// errors so it can run inline on the signin path.
//
// RULE: it MUST NOT change the stored password. A failed-login lockout is triggered
// by unauthenticated input against a known username, so resetting the credential
// here would hand an attacker a one-request account-takeover-disruption primitive
// (the original CWE-645 bug). The lock is temporary and clears on its own; the owner
// signs in normally once the window passes.
func (service *User) NotifySigninLockout(session data.Session, username string) {

	const location = "service.User.NotifySigninLockout"

	user := model.NewUser()
	if err := service.LoadByUsername(session, username, &user); err != nil {

		// A failed attempt against a username that does not exist has no owner to
		// notify. This is an expected, high-volume case (attackers guess nonexistent
		// usernames), so do not report it -- only report genuine load errors.
		if !derp.IsNotFound(err) {
			derp.Report(derp.Wrap(err, location, "Loading user by username", username))
		}

		return
	}

	// Notify the owner that their account is being targeted. No password reset code
	// is issued -- the credential is untouched.
	if err := service.emailService.SendUserLockout(session, &user); err != nil {
		derp.Report(derp.Wrap(err, location, "Sending user lockout email", user))
	}
}

// MakeNewPasswordResetCode generates a new password reset code (valid for the provided duration) for the provided user.
func (service *User) MakeNewPasswordResetCode(session data.Session, user *model.User, duration time.Duration) error {

	// If the PasswordReset IS NOT active then
	// create a new password reset code for this user
	if user.PasswordReset.NotActive() {
		user.PasswordReset = model.NewPasswordReset(duration)
	}

	// In all cases, refresh the expiration date of the password reset code
	// so that overlapping requests all share the most recent deadline.
	user.PasswordReset.RefreshExpireDate(duration)

	// Try to save the user with the new password reset code.
	if err := service.Save(session, user, "Create Password Reset Code"); err != nil {
		return derp.Wrap(err, "service.User.MakeNewPasswordResetCode", "Saving user", user)
	}

	return nil
}

/******************************************
 * WebFinger Behavior
 ******************************************/

// WebFinger returns the WebFinger resource for the User identified by the provided token
func (service *User) WebFinger(session data.Session, token string) (digit.Resource, error) {

	const location = "service.User.WebFinger"

	// Try to load the user from the database
	user := model.NewUser()
	if err := service.LoadByToken(session, token, &user); err != nil {
		return digit.Resource{}, derp.Wrap(err, location, "Loading user", token)
	}

	// RULE: Non-public profiles are hidden from public discovery. WebFinger is unauthenticated,
	// so there is no requester to exempt (the owner discovers themselves via the app, not WebFinger).
	// This keeps "Hidden from Public Servers" true at the discovery layer, matching the hidden
	// actor document and every sibling ActivityPub endpoint.
	if !user.IsPublic {
		return digit.Resource{}, derp.NotFound(location, "User not found", token)
	}

	// Make a WebFinger resource for this user.
	result := digit.NewResource("acct:"+user.Username+"@"+uri.Hostname(service.host)).
		Alias(service.host+"/@"+user.Username).
		Alias(service.host+"/@"+user.UserID.Hex()).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, user.ActivityPubURL()).
		Link(digit.RelationTypeHub, model.MimeTypeJSONFeed, user.JSONFeedURL()).
		Link(digit.RelationTypeProfile, model.MimeTypeHTML, user.ActivityPubURL()).
		Link(digit.RelationTypeAvatar, model.MimeTypeImage, user.ActivityPubIconURL()).
		Link(digit.RelationTypeSubscribeRequest, "", service.RemoteFollowURL()).
		Link(digit.RelationTypeOpenIDIssuer, "", service.host).
		Link(camper.IntentTypeCreate, "", service.CreateIntentURL()).
		Link(camper.IntentTypeDislike, "", service.DislikeIntentURL()).
		Link(camper.IntentTypeFollow, "", service.FollowIntentURL()).
		Link(camper.IntentTypeLike, "", service.LikeIntentURL())

	return result, nil
}

// RemoteFollowURL returns the URL template for this server's remote-follow ("subscribe request") endpoint
func (service *User) RemoteFollowURL() string {
	return service.host + "/@me/settings/following-edit?url={uri}"
}

// CreateIntentURL returns the URL template for this server's "create" Activity Intent endpoint
func (service *User) CreateIntentURL() string {
	return service.host + "/@me/intent/create?type={type}&name={name}&summary={summary}&content={content}&inReplyTo={inReplyTo}&on-success={on-success}&on-cancel={on-cancel}"
}

// DislikeIntentURL returns the URL template for this server's "dislike" Activity Intent endpoint
func (service *User) DislikeIntentURL() string {
	return service.host + "/@me/intent/dislike?object={object}&on-success={on-success}&on-cancel={on-cancel}"
}

// FollowIntentURL returns the URL template for this server's "follow" Activity Intent endpoint
func (service *User) FollowIntentURL() string {
	return service.host + "/@me/intent/follow?object={object}&on-success={on-success}&on-cancel={on-cancel}"
}

// LikeIntentURL returns the URL template for this server's "like" Activity Intent endpoint
func (service *User) LikeIntentURL() string {
	return service.host + "/@me/intent/like?object={object}&on-success={on-success}&on-cancel={on-cancel}"
}

// CalculateTags scans the Template-defined TagPaths on this User for #hashtags,
// then writes the normalized tag names back onto the User record.
func (service *User) CalculateTags(session data.Session, user *model.User) {

	const location = "service.User.CalculateTags"

	// Load the Template (to get TagPaths)
	template, err := service.templateService.Load(user.OutboxTemplate)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading template", user.OutboxTemplate))
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
	hashtagNames, _, err := service.searchTagService.NormalizeTags(session, hashtags...)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Normalizing tags", hashtags))
	}

	// Apply the normalized hashtag names to the user object
	user.Hashtags = hashtagNames
}

/******************************************
 * SearchResulter Interface
 ******************************************/

// SearchResult returns the search index entry for the provided User: a live record for
// public+indexable Users, or a delete marker for everyone else.
func (service *User) SearchResult(user *model.User) model.SearchResult {

	result := model.NewSearchResult()

	if user.IsPublic && user.IsIndexable {

		result.Type = "Person"
		result.Name = user.DisplayName
		result.AttributedTo = "@" + user.Username
		result.Summary = user.StatusMessage
		result.URL = user.ProfileURL
		result.IconURL = user.ActivityPubIconURL()
		result.Tags = user.Hashtags
		result.Text = user.DisplayName + " " + user.Username + " " + strings.Join(user.Hashtags, " ")
		result.Local = true

		return result
	}

	result.URL = user.ProfileURL
	result.DeleteDate = time.Now().Unix()

	return result

}
