package build

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User is a builder for the admin/users page
// It can only be accessed by a Domain Owner
type User struct {
	_user *model.User
	CommonWithTemplate
}

// NewUser returns a fully initialized `User` builder.
func NewUser(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, template model.Template, user *model.User, actionID string) (User, error) {

	const location = "build.NewUser"

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, user, actionID)

	if err != nil {
		return User{}, derp.Wrap(err, location, "Creating common builder")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return User{}, derp.Forbidden(location, "Must be domain owner to continue")
	}

	// Return the User builder
	return User{
		_user:              user,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this User
func (w User) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.User.Render", "Generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this User
func (w User) View(actionID string) (template.HTML, error) {

	builder, err := NewUser(w._factory, w._session, w._request, w._response, w._template, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.User.View", "Creating builder")
	}

	return builder.Render()
}

// NavigationID returns the top-level navigation item to highlight. Implements the Builder interface.
func (w User) NavigationID() string {
	return "admin"
}

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w User) Token() string {
	return "users"
}

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w User) PageTitle() string {
	return "Settings"
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w User) Permalink() string {
	return w.Host() + "/admin/users/" + w.UserID()
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w User) BasePath() string {
	return "/admin/users/" + w.UserID()
}

// object returns the model object being built. Implements the Builder interface.
func (w User) object() data.Object {
	return w._user
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w User) objectID() primitive.ObjectID {
	return w._user.UserID
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w User) objectType() string {
	return "User"
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w User) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w User) service() service.ModelService {
	return w._factory.User()
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
func (w User) clone(action string) (Builder, error) {
	return NewUser(w._factory, w._session, w._request, w._response, w._template, w._user, action)
}

// setState makes this builder a StateSetter so template pipelines can run a `set-state` step
// against an admin-managed User (mirrors Outbox.setState). FIX #2 relies on this to move a
// newly created User to the "LIVE" state during the admin "add" action.
func (w User) setState(stateID string) error {
	w._user.SetState(stateID)
	return nil
}

/******************************************
 * User Data
 ******************************************/

// UserID returns the unique ID of the User being built, as a string
func (w User) UserID() string {
	if w._user == nil {
		return ""
	}
	return w._user.UserID.Hex()
}

// ProfileURL returns the profile URL of the User being built
func (w User) ProfileURL() string {
	return w._user.ProfileURL
}

// Label returns the label of the User being built
func (w User) Label() string {
	if w._user == nil {
		return ""
	}
	return w._user.DisplayName
}

// DisplayName returns the display name of the User being built
func (w User) DisplayName() string {
	if w._user == nil {
		return ""
	}
	return w._user.DisplayName
}

// IconURL returns the icon URL of the User being built
func (w User) IconURL() string {
	if w._user == nil {
		return ""
	}
	return w._user.ActivityPubIconURL()
}

// MapIDs returns the map IDs of the User being built
func (w User) MapIDs() map[string]string {
	return w._user.MapIDs
}

/******************************************
 * Query Builders
 ******************************************/

// Users returns a QueryBuilder that lists Users
func (w User) Users() *QueryBuilder[model.UserSummary] {

	query := builder.NewBuilder().
		String("search", builder.WithAlias("displayName"), builder.WithDefaultOpContains()).
		String("displayName").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.UserSummary](w._factory.User(), w._session, criteria)

	return &result
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IsAdminBuilder returns TRUE because User is an admin route.
func (w User) IsAdminBuilder() bool {
	return true
}

// Groups returns a slice of all Groups in the database
func (w User) Groups() ([]form.LookupCode, error) {
	groupService := w._factory.Group()
	groups, err := groupService.Query(w._session, exp.All(), option.SortAsc("label"))

	if err != nil {
		return nil, derp.Wrap(err, "build.User.Groups", "Loading groups")
	}

	result := slice.Map(groups, func(group model.Group) form.LookupCode {
		return form.LookupCode{
			Value: group.GroupID.Hex(),
			Label: group.Label,
		}
	})

	return result, nil
}

// CountUsers returns the total number of users in the database
func (w User) CountUsers() (int64, error) {
	return w._factory.User().Count(w._session, exp.All())
}

// CountPublicUsers returns the total number of users marked "isPublic"
func (w User) CountPublicUsers() (int64, error) {
	return w._factory.User().Count(w._session, exp.Equal("isPublic", true))
}

// CountIndexableUsers returns the total number of users marked "isIndexable"
func (w User) CountIndexableUsers() (int64, error) {
	return w._factory.User().Count(w._session, exp.And(exp.Equal("isPublic", true), exp.Equal("isIndexable", true)))
}

// Registration returns the signup template selected for this domain
func (w User) Registration() model.Registration {

	if domain := w._factory.Domain().Get(); domain.RegistrationID != "" {
		if template, err := w._factory.Registration().Load(domain.RegistrationID); err == nil {
			return template
		}
	}

	return model.NewRegistration("", nil)
}

// AssignedGroups lists all groups to which the current user is assigned.
func (w User) AssignedGroups() ([]model.Group, error) {
	groupService := w._factory.Group()
	result, err := groupService.ListByIDs(w._session, w._user.GroupIDs...)

	return result, derp.Wrap(err, "build.User.AssignedGroups", "Listing groups", w._user.GroupIDs)
}

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w User) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_users")
}
