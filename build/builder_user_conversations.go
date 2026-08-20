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
		return Conversations{}, derp.Wrap(err, location, "Loading template")
	}

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, user, actionID)

	if err != nil {
		return Conversations{}, derp.Wrap(err, location, "Creating common builder")
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
		return "", derp.Wrap(status.Error, "build.Conversations.Render", "Generating HTML", w._request.URL.String())
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Conversations
func (w Conversations) View(actionID string) (template.HTML, error) {

	builder, err := NewConversations(w._factory, w._session, w._request, w._response, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.Conversations.View", "Creating Conversations builder")
	}

	return builder.Render()
}

// NavigationID returns the ID to use for highlighing navigation menus
func (w Conversations) NavigationID() string {
	return "conversations"
}

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w Conversations) PageTitle() string {
	return w._user.DisplayName
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w Conversations) BasePath() string {
	return "/@me/conversations"
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w Conversations) Permalink() string {
	return w.Host() + "/@me/conversations"
}

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w Conversations) Token() string {
	return "conversations"
}

// object returns the model object being built. Implements the Builder interface.
func (w Conversations) object() data.Object {
	return w._user
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w Conversations) objectID() primitive.ObjectID {
	return w._user.UserID
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w Conversations) objectType() string {
	return "User"
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w Conversations) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w Conversations) service() service.ModelService {
	return w._factory.User()
}

// templateRole returns the role this Template plays, which chooses the child template. Implements the Builder interface.
func (w Conversations) templateRole() string {
	return "conversations"
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
func (w Conversations) clone(action string) (Builder, error) {
	return NewConversations(w._factory, w._session, w._request, w._response, w._user, action)
}

/******************************************
 * Data Accessors
 ******************************************/

// UserID returns the unique ID of the User whose conversations are being built, as a string
func (w Conversations) UserID() string {
	return w._user.UserID.Hex()
}

// ActorURL returns the ActivityPub URL of the User whose conversations are being built
func (w Conversations) ActorURL() string {
	return w.Host() + "/@" + w._user.UserID.Hex()
}

// Myself returns TRUE if the current user is viewing their own profile
func (w Conversations) Myself() bool {
	return w._authorization.UserID == w._user.UserID
}

// Username returns the username of the Conversations being built
func (w Conversations) Username() string {
	return w._user.Username
}

// DisplayName returns the display name of the Conversations being built
func (w Conversations) DisplayName() string {
	return w._user.DisplayName
}

// ProfileURL returns the profile URL of the Conversations being built
func (w Conversations) ProfileURL() string {
	return w._user.ProfileURL
}

// IconURL returns the icon URL of the Conversations being built
func (w Conversations) IconURL() string {
	return w._user.ActivityPubIconURL()
}

/******************************************
 * Conversations Methods
 ******************************************/

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w Conversations) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Conversations")
}
