package build

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Outbox builds individual messages from a User's Outbox.
type Outbox struct {
	_user *model.User
	CommonWithTemplate
}

// NewOutbox returns a fully initialized `Outbox` builder.
func NewOutbox(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, user *model.User, actionID string) (Outbox, error) {

	const location = "build.NewOutbox"

	// Load the Template
	templateService := factory.Template()
	template, err := templateService.Load(user.OutboxTemplate) // Users should get to choose their own outbox template

	if err != nil {
		return Outbox{}, derp.Wrap(err, location, "Loading template")
	}

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, user, actionID)

	if err != nil {
		return Outbox{}, derp.Wrap(err, location, "Creating common builder")
	}

	// Verify that the User's profile is visible
	if !isUserVisible(&common._authorization, user) {
		return Outbox{}, derp.NotFound(location, "User not found")
	}

	// Enforce user permissions on the requested action
	if !common.UserCan(actionID) {
		if common._authorization.IsAuthenticated() {
			return Outbox{}, derp.Forbidden(location, "Forbidden (signed in as user)")
		} else if common._authorization.IsIdentity() {
			return Outbox{}, derp.Forbidden(location, "Forbidden (signed in as guest)")
		} else {
			return Outbox{}, derp.Unauthorized(location, "Anonymous user is not authorized to perform this action", user.ProfileURL, actionID)
		}
	}

	// Return the Outbox builder
	return Outbox{
		_user:              user,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Outbox
func (w Outbox) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Outbox.Render", "Generating HTML", w._request.URL.String())
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Outbox
func (w Outbox) View(actionID string) (template.HTML, error) {

	builder, err := NewOutbox(w._factory, w._session, w._request, w._response, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.Outbox.View", "Creating Outbox builder")
	}

	return builder.Render()
}

// NavigationID returns the ID to use for highlighing navigation menus
func (w Outbox) NavigationID() string {
	if w._user.UserID == w.AuthenticatedID() {
		return "outbox"
	}
	return "user"
}

// PageTitle returns the User's display name, for use in the page <title>
func (w Outbox) PageTitle() string {
	return w._user.DisplayName
}

// Permalink returns the absolute, canonical URL of this User's profile
func (w Outbox) Permalink() string {
	return w.Host() + "/@" + w._user.UserID.Hex()
}

// BasePath returns the root path that this builder's actions hang from
func (w Outbox) BasePath() string {
	return "/@" + w._user.UserID.Hex()
}

// Token returns the name of this builder's route family
func (w Outbox) Token() string {
	return "users"
}

// object returns the User record that this builder wraps
func (w Outbox) object() data.Object {
	return w._user
}

// objectID returns the UserID of the record that this builder wraps
func (w Outbox) objectID() primitive.ObjectID {
	return w._user.UserID
}

// objectType returns the name of the model that this builder wraps
func (w Outbox) objectType() string {
	return "User"
}

// schema returns the JSON-Schema that validates writes to this builder's object
func (w Outbox) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

// service returns the ModelService that loads and saves this builder's object
func (w Outbox) service() service.ModelService {
	return w._factory.User()
}

// templateRole returns the Template role that this builder can build
func (w Outbox) templateRole() string {
	return "outbox"
}

// clone returns a copy of this builder, bound to a different action
func (w Outbox) clone(action string) (Builder, error) {
	return NewOutbox(w._factory, w._session, w._request, w._response, w._user, action)
}

// IsMyself returns TRUE if the outbox record is owned
// by the currently signed-in user
func (w Outbox) IsMyself() bool {
	return w._user.IsMyself(w._authorization.UserID)
}

/******************************************
 * Data Accessors
 ******************************************/

// UserID returns the hex-encoded ID of the User being built
func (w Outbox) UserID() string {
	return w._user.UserID.Hex()
}

// Myself returns TRUE if the current user is viewing their own profile
func (w Outbox) Myself() bool {
	return w._user.IsMyself(w._authorization.UserID)
}

// Username returns the username of the User being built
func (w Outbox) Username() string {
	return w._user.Username
}

// RuleCount returns the number of Rules that the User has defined
func (w Outbox) RuleCount() int {
	return w._user.RuleCount
}

// FollowerCount returns the number of Followers that the User has
func (w Outbox) FollowerCount() int {
	return w._user.FollowerCount
}

// FollowingCount returns the number of Actors that the User follows
func (w Outbox) FollowingCount() int {
	return w._user.FollowingCount
}

// DisplayName returns the display name of the User being built
func (w Outbox) DisplayName() string {
	return w._user.DisplayName
}

// StateID returns the workflow state of the User being built
func (w Outbox) StateID() string {
	return w._user.StateID
}

// IsPublic returns TRUE if this User's profile is visible to anonymous visitors
func (w Outbox) IsPublic() bool {
	return w._user.IsPublic
}

// IsIndexable returns TRUE if this user's profile may be indexed by search
// engines. It reflects the profile owner's "Index on Search Engines" setting
// and overrides Common.IsIndexable so shared page templates can emit a
// "noindex" robots directive when the owner has opted out.
func (w Outbox) IsIndexable() bool {
	return w._user.IsIndexable
}

// IsBridgeBluesky returns TRUE if this user is bridged to ATProtocol (Bluesky)
func (w Outbox) IsBridgeBluesky() bool {
	return w._user.IsBridgeBluesky.Bool()
}

// BlueskyHandle returns the ATProtocol handle for this user, if they are bridged to Bluesky
func (w Outbox) BlueskyHandle() string {
	return w._user.Username + "." + w.Hostname()
}

// StatusMessage returns the profile summary in plain text, with #hashtags linkified.
func (w Outbox) StatusMessage() string {
	return w._user.StatusMessage
}

// StatusMessageHTML returns the profile summary rendered from Markdown, with #hashtags linkified.
func (w Outbox) StatusMessageHTML() template.HTML {
	return template.HTML(w._user.SummaryHTML())
}

// ProfileURL returns the User's canonical profile URL
func (w Outbox) ProfileURL() string {
	return w._user.ProfileURL
}

// IconURL returns the URL of the User's avatar image
func (w Outbox) IconURL() string {
	return w._user.ActivityPubIconURL()
}

// ImageURL returns the URL of the User's banner image
func (w Outbox) ImageURL() string {
	return w._user.ActivityPubImageURL()
}

// Location returns the User's self-reported location
func (w Outbox) Location() string {
	return w._user.Location
}

// Links returns the User's list of profile links
func (w Outbox) Links() sliceof.Object[model.PersonLink] {
	return w._user.Links
}

// Tags returns all tags (mentions, hashtags, etc) for the user being built
func (w Outbox) Tags() sliceof.Object[mapof.String] {
	return slice.Map(w._user.Hashtags, func(tag string) mapof.String {
		return mapof.String{
			"Name": tag,
			"Type": vocab.LinkTypeHashtag,
			"Href": model.HashtagURL(w.Host(), w._user.TagURL, tag),
		}
	})
}

// Data returns a single custom value from the User's extra data map
func (w Outbox) Data(path string) any {
	return w._user.Data[path]
}

// OEmbedJSON returns the URL for the oEmbed JSON endpoint for this User's profile
func (w Outbox) OEmbedJSON() string {
	return oEmbedURL(w.Host(), w.Permalink(), "json")
}

// OEmbedXML returns the URL for the oEmbed XML endpoint for this User's profile
func (w Outbox) OEmbedXML() string {
	return oEmbedURL(w.Host(), w.Permalink(), "xml")
}

// ActivityPubURL returns the URL of the User's ActivityPub actor document
func (w Outbox) ActivityPubURL() string {
	return w._user.ActivityPubURL()
}

// ActivityPubIconURL returns the URL of the User's avatar, as published to ActivityPub
func (w Outbox) ActivityPubIconURL() string {
	return w._user.ActivityPubIconURL()
}

// ActivityPubInboxURL returns the URL of the User's ActivityPub inbox
func (w Outbox) ActivityPubInboxURL() string {
	return w._user.ActivityPubInboxURL()
}

// ActivityPubOutboxURL returns the URL of the User's ActivityPub outbox
func (w Outbox) ActivityPubOutboxURL() string {
	return w._user.ActivityPubOutboxURL()
}

// ActivityPubFollowersURL returns the URL of the User's ActivityPub followers collection
func (w Outbox) ActivityPubFollowersURL() string {
	return w._user.ActivityPubFollowersURL()
}

// ActivityPubFollowingURL returns the URL of the User's ActivityPub following collection
func (w Outbox) ActivityPubFollowingURL() string {
	return w._user.ActivityPubFollowingURL()
}

// ActivityPubLikedURL returns the URL of the User's ActivityPub liked collection
func (w Outbox) ActivityPubLikedURL() string {
	return w._user.ActivityPubLikedURL()
}

// ActivityPubPublicKeyURL returns the URL of the User's ActivityPub public key
func (w Outbox) ActivityPubPublicKeyURL() string {
	return w._user.ActivityPubPublicKeyURL()
}

/******************************************
 * Outbox Methods
 ******************************************/

// Outbox returns a QueryBuilder for the User's own top-level Streams
func (w Outbox) Outbox() QueryBuilder[model.StreamSummary] {

	expressionBuilder := builder.NewBuilder().
		Int("publishDate")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("parentId", w._user.UserID),
		exp.Equal("inReplyTo", ""),
		w.defaultAllowed(),
	)

	result := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), w._session, criteria)

	return result
}

// Circles returns a QueryBuilder for the Circles that the User has defined
func (w Outbox) Circles() QueryBuilder[model.Circle] {

	expressionBuilder := builder.NewBuilder().
		String("name")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w.objectID()),
	)

	result := NewQueryBuilder[model.Circle](w._factory.Circle(), w._session, criteria)

	return result
}

// HasProducts returns TRUE if the User has any purchaseable Products
func (w Outbox) HasProducts() (bool, error) {
	return w._factory.Circle().HasProducts(w._session, w._user.UserID)
}

// ProductCount returns the number of purchaseable Products that the User offers
func (w Outbox) ProductCount() (int, error) {
	return w._factory.Circle().ProductCount(w._session, w._user.UserID)
}

// Products returns every purchaseable Product from the User's Featured Circles
func (w Outbox) Products() (sliceof.Object[model.Product], error) {

	const location = "build.Outbox.Products"

	// Get purchaseable products from all Featured Circles
	productIDs, err := w._factory.Circle().AssignedProductIDs(w._session, w._user.UserID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Retrieving remote products for user", w._user.UserID.Hex())
	}

	// If there are no remote products, return an empty slice
	if productIDs.IsEmpty() {
		return sliceof.Object[model.Product]{}, nil
	}

	// Look up the products for this User using their IDs
	products, err := w._factory.Product().QueryByIDs(w._session, w._user.UserID, productIDs...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Retrieving remote products for user", w._user.UserID.Hex())
	}

	return products, nil
}

// Replies returns a QueryBuilder for the User's Streams that reply to something else
func (w Outbox) Replies() QueryBuilder[model.StreamSummary] {

	expressionBuilder := builder.NewBuilder().
		Int("publishDate")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("parentId", w._user.UserID),
		exp.NotEqual("inReplyTo", ""),
		w.defaultAllowed(),
	)

	result := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), w._session, criteria)

	return result
}

// Responses returns a QueryBuilder for the User's Responses (likes, dislikes, and shares)
func (w Outbox) Responses() QueryBuilder[model.Response] {

	expressionBuilder := builder.NewBuilder().
		Int("createDate")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w.objectID()),
	)

	result := NewQueryBuilder[model.Response](w._factory.Response(), w._session, criteria)

	return result
}

// setState moves the User into a different workflow state
func (w Outbox) setState(stateID string) error {
	w._user.SetState(stateID)
	return nil
}

// debug writes the wrapped User to the debug log
func (w Outbox) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Outbox")
}
