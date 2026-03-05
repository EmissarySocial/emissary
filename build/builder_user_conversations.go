package build

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Conversations is a builder for the @user/inbox page
type Conversations struct {
	_user *model.User
	CommonWithTemplate
}

// NewConversations returns a fully initialized `Conversations` builder
func NewConversations(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, user *model.User, actionID string) (Conversations, error) {

	const location = "build.NewConversations"

	// Load the Template
	templateService := factory.Template()
	template, err := templateService.Load("user-conversations")

	if err != nil {
		return Conversations{}, derp.Wrap(err, location, "Unable to load template")
	}

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, user, actionID)

	if err != nil {
		return Conversations{}, derp.Wrap(err, location, "Unable to create common builder")
	}

	// Enforce user permissions on the requested action
	if !common.UserCan(actionID) {
		if common._authorization.IsAuthenticated() {
			return Conversations{}, derp.Forbidden(location, "Forbidden", "User is authenticated, but this action is not allowed", actionID)
		} else {
			return Conversations{}, derp.Unauthorized(location, "Anonymous user is not authorized to perform this action", user.ProfileURL, actionID)
		}
	}

	return Conversations{
		_user:              user,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Conversations
func (w Conversations) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		return "", derp.Wrap(status.Error, "build.Conversations.Render", "Unable to generate HTML", w._request.URL.String())
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Conversations
func (w Conversations) View(actionID string) (template.HTML, error) {

	builder, err := NewConversations(w._factory, w._session, w._request, w._response, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.Conversations.View", "Unable to create Conversations builder")
	}

	return builder.Render()
}

// NavigationID returns the ID to use for highlighing navigation menus
func (w Conversations) NavigationID() string {
	return "conversations"
}

func (w Conversations) PageTitle() string {
	return w._user.DisplayName
}

func (w Conversations) BasePath() string {
	return "/@me/conversations"
}

func (w Conversations) Permalink() string {
	return w.Host() + "/@me/conversations"
}

func (w Conversations) Token() string {
	return "conversations"
}

func (w Conversations) object() data.Object {
	return w._user
}

func (w Conversations) objectID() primitive.ObjectID {
	return w._user.UserID
}

func (w Conversations) objectType() string {
	return "User"
}

func (w Conversations) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w Conversations) service() service.ModelService {
	return w._factory.User()
}

func (w Conversations) templateRole() string {
	return "conversations"
}

func (w Conversations) clone(action string) (Builder, error) {
	return NewConversations(w._factory, w._session, w._request, w._response, w._user, action)
}

/******************************************
 * Data Accessors
 ******************************************/

func (w Conversations) UserID() string {
	return w._user.UserID.Hex()
}

func (w Conversations) ActorURL() string {
	return w.Host() + "/@" + w._user.UserID.Hex()
}

// Myself returns TRUE if the current user is viewing their own profile
func (w Conversations) Myself() bool {
	return w._authorization.UserID == w._user.UserID
}

func (w Conversations) Username() string {
	return w._user.Username
}

func (w Conversations) DisplayName() string {
	return w._user.DisplayName
}

func (w Conversations) ProfileURL() string {
	return w._user.ProfileURL
}

func (w Conversations) IconURL() string {
	return w._user.ActivityPubIconURL()
}

/******************************************
 * Conversations Methods
 ******************************************/

func (w Conversations) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Conversations")
}
