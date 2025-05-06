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
func NewUser(factory Factory, request *http.Request, response http.ResponseWriter, template model.Template, user *model.User, actionID string) (User, error) {

	const location = "build.NewUser"

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, request, response, template, actionID)

	if err != nil {
		return User{}, derp.Wrap(err, location, "Error creating common builder")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return User{}, derp.ForbiddenError(location, "Must be domain owner to continue")
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
	status := Pipeline(w._action.Steps).Get(w.factory(), &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.User.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this User
func (w User) View(actionID string) (template.HTML, error) {

	builder, err := NewUser(w._factory, w._request, w._response, w._template, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.User.View", "Error creating builder")
	}

	return builder.Render()
}

func (w User) NavigationID() string {
	return "admin"
}

func (w User) Token() string {
	return "users"
}

func (w User) PageTitle() string {
	return "Settings"
}

func (w User) Permalink() string {
	return w.Host() + "/admin/users/" + w.UserID()
}

func (w User) BasePath() string {
	return "/admin/users/" + w.UserID()
}

func (w User) object() data.Object {
	return w._user
}

func (w User) objectID() primitive.ObjectID {
	return w._user.UserID
}

func (w User) objectType() string {
	return "User"
}

func (w User) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w User) service() service.ModelService {
	return w._factory.User()
}

func (w User) clone(action string) (Builder, error) {
	return NewUser(w._factory, w._request, w._response, w._template, w._user, action)
}

/******************************************
 * User Data
 ******************************************/

func (w User) UserID() string {
	if w._user == nil {
		return ""
	}
	return w._user.UserID.Hex()
}

func (w User) ProfileURL() string {
	return w._user.ProfileURL
}

func (w User) Label() string {
	if w._user == nil {
		return ""
	}
	return w._user.DisplayName
}

func (w User) DisplayName() string {
	if w._user == nil {
		return ""
	}
	return w._user.DisplayName
}

func (w User) IconURL() string {
	if w._user == nil {
		return ""
	}
	return w._user.ActivityPubIconURL()
}

func (w User) MapIDs() map[string]string {
	return w._user.MapIDs
}

/******************************************
 * Query Builders
 ******************************************/

func (w User) Users() *QueryBuilder[model.UserSummary] {

	query := builder.NewBuilder().
		String("search", builder.WithAlias("displayName"), builder.WithDefaultOpContains()).
		String("displayName").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.UserSummary](w._factory.User(), criteria)

	return &result
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IsAdminBuilder returns TRUE because User is an admin route.
func (w User) IsAdminBuilder() bool {
	return false
}

// Groups returns a slice of all Groups in the database
func (w User) Groups() ([]form.LookupCode, error) {
	groupService := w.factory().Group()
	groups, err := groupService.Query(exp.All(), option.SortAsc("label"))

	if err != nil {
		return nil, derp.Wrap(err, "build.User.Groups", "Error loading groups")
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
	return w.factory().User().Count(exp.All())
}

// CountPublicUsers returns the total number of users marked "isPublic"
func (w User) CountPublicUsers() (int64, error) {
	return w.factory().User().Count(exp.Equal("isPublic", true))
}

// CountIndexableUsers returns the total number of users marked "isIndexable"
func (w User) CountIndexableUsers() (int64, error) {
	return w.factory().User().Count(exp.And(exp.Equal("isPublic", true), exp.Equal("isIndexable", true)))
}

// Registration returns the signup template selected for this domain
func (w User) Registration() model.Registration {

	domain := w._factory.Domain().Get()

	if domain.RegistrationID != "" {
		if template, err := w._factory.Registration().Load(domain.RegistrationID); err == nil {
			return template
		}
	}

	return model.NewRegistration("", nil)
}

// AssignedGroups lists all groups to which the current user is assigned.
func (w User) AssignedGroups() ([]model.Group, error) {
	groupService := w.factory().Group()
	result, err := groupService.ListByIDs(w._user.GroupIDs...)

	return result, derp.Wrap(err, "build.User.AssignedGroups", "Error listing groups", w._user.GroupIDs)
}

func (w User) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_users")
}
