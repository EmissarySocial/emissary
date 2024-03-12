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
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Outbox builds individual messages from a User's Outbox.
type Outbox struct {
	_user *model.User
	Common
}

// NewOutbox returns a fully initialized `Outbox` builder.
func NewOutbox(factory Factory, request *http.Request, response http.ResponseWriter, user *model.User, actionID string) (Outbox, error) {

	// Load the Template
	templateService := factory.Template()
	template, err := templateService.Load("user-outbox") // Users should get to choose their own outbox template

	if err != nil {
		return Outbox{}, derp.Wrap(err, "build.NewOutbox", "Error loading template")
	}

	// Create the underlying Common builder
	common, err := NewCommon(factory, request, response, template, actionID)

	if err != nil {
		return Outbox{}, derp.Wrap(err, "build.NewOutbox", "Error creating common builder")
	}

	// Verify that the User's profile is visible
	if !isUserVisible(&common._authorization, user) {
		return Outbox{}, derp.NewNotFoundError("build.NewOutbox", "User not found")
	}

	// Return the Outbox builder
	return Outbox{
		_user:  user,
		Common: common,
	}, nil
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this Outbox
func (w Outbox) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w.action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Outbox.Render", "Error generating HTML", w._request.URL.String())
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Outbox
func (w Outbox) View(actionID string) (template.HTML, error) {

	builder, err := NewOutbox(w._factory, w._request, w._response, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.Outbox.View", "Error creating Outbox builder")
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

func (w Outbox) PageTitle() string {
	return w._user.DisplayName
}

func (w Outbox) Permalink() string {
	return w.Host() + "/@" + w._user.UserID.Hex()
}

func (w Outbox) BasePath() string {
	return "/@" + w._user.UserID.Hex()
}

func (w Outbox) Token() string {
	return "users"
}

func (w Outbox) object() data.Object {
	return w._user
}

func (w Outbox) objectID() primitive.ObjectID {
	return w._user.UserID
}

func (w Outbox) objectType() string {
	return "User"
}

func (w Outbox) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w Outbox) service() service.ModelService {
	return w._factory.User()
}

func (w Outbox) templateRole() string {
	return "outbox"
}

func (w Outbox) clone(action string) (Builder, error) {
	return NewOutbox(w._factory, w._request, w._response, w._user, action)
}

// UserCan returns TRUE if this Request is authorized to access the requested view
func (w Outbox) UserCan(actionID string) bool {

	action, ok := w._template.Action(actionID)

	if !ok {
		return false
	}

	authorization := w.authorization()

	return action.UserCan(w._user, &authorization)
}

// IsMyself returns TRUE if the outbox record is owned
// by the currently signed-in user
func (w Outbox) IsMyself() bool {
	return w._user.UserID == w.authorization().UserID
}

/******************************************
 * Data Accessors
 ******************************************/

func (w Outbox) UserID() string {
	return w._user.UserID.Hex()
}

// Myself returns TRUE if the current user is viewing their own profile
func (w Outbox) Myself() bool {
	return w._authorization.UserID == w._user.UserID
}

func (w Outbox) Username() string {
	return w._user.Username
}

func (w Outbox) RuleCount() int {
	return w._user.RuleCount
}

func (w Outbox) FollowerCount() int {
	return w._user.FollowerCount
}

func (w Outbox) FollowingCount() int {
	return w._user.FollowingCount
}

func (w Outbox) DisplayName() string {
	return w._user.DisplayName
}

func (w Outbox) StatusMessage() string {
	return w._user.StatusMessage
}

func (w Outbox) ProfileURL() string {
	return w._user.ProfileURL
}

func (w Outbox) ImageURL() string {
	return w._user.ActivityPubAvatarURL()
}

func (w Outbox) Location() string {
	return w._user.Location
}

func (w Outbox) Links() []model.PersonLink {
	return w._user.Links
}

func (w Outbox) ActivityPubURL() string {
	return w._user.ActivityPubURL()
}

func (w Outbox) ActivityPubAvatarURL() string {
	return w._user.ActivityPubAvatarURL()
}

func (w Outbox) ActivityPubInboxURL() string {
	return w._user.ActivityPubInboxURL()
}

func (w Outbox) ActivityPubOutboxURL() string {
	return w._user.ActivityPubOutboxURL()
}

func (w Outbox) ActivityPubFollowersURL() string {
	return w._user.ActivityPubFollowersURL()
}

func (w Outbox) ActivityPubFollowingURL() string {
	return w._user.ActivityPubFollowingURL()
}

func (w Outbox) ActivityPubLikedURL() string {
	return w._user.ActivityPubLikedURL()
}

func (w Outbox) ActivityPubPublicKeyURL() string {
	return w._user.ActivityPubPublicKeyURL()
}

/******************************************
 * Outbox Methods
 ******************************************/

func (w Outbox) Outbox() QueryBuilder[model.StreamSummary] {

	expressionBuilder := builder.NewBuilder().
		Int("publishDate")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("parentId", w._user.UserID),
		exp.Equal("inReplyTo", ""),
	)

	result := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), criteria)

	return result
}

func (w Outbox) Replies() QueryBuilder[model.StreamSummary] {

	expressionBuilder := builder.NewBuilder().
		Int("publishDate")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("parentId", w._user.UserID),
		exp.NotEqual("inReplyTo", ""),
	)

	result := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), criteria)

	return result
}

func (w Outbox) Responses() QueryBuilder[model.Response] {

	expressionBuilder := builder.NewBuilder().
		Int("createDate")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w.objectID()),
	)

	result := NewQueryBuilder[model.Response](w._factory.Response(), criteria)

	return result
}

func (w Outbox) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Outbox")
}
