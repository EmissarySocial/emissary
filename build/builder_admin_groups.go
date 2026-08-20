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

// Group is a builder for the admin/groups page
// It can only be accessed by a Domain Owner
type Group struct {
	_group *model.Group
	CommonWithTemplate
}

// NewGroup returns a fully initialized `Group` builder.
func NewGroup(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, template model.Template, group *model.Group, actionID string) (Group, error) {

	const location = "build.NewGroup"

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, group, actionID)

	if err != nil {
		return Group{}, derp.Wrap(err, location, "Creating common builder")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return Group{}, derp.Forbidden(location, "Must be domain owner to continue")
	}

	// Return the Group builder
	return Group{
		_group:             group,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Stream
func (w Group) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Group.Render", "Generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Group
func (w Group) View(actionID string) (template.HTML, error) {

	const location = "build.Group.View"

	builder, err := NewGroup(w._factory, w._session, w._request, w._response, w._template, w._group, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Creating Group builder")
	}

	return builder.Render()
}

// NavigationID returns the top-level navigation item to highlight. Implements the Builder interface.
func (w Group) NavigationID() string {
	return "admin"
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w Group) Permalink() string {
	return w.Host() + "/groups/" + w.GroupID()
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w Group) BasePath() string {
	if w._group == nil {
		return "/groups"
	}
	return "/groups/" + w.GroupID()
}

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w Group) Token() string {
	return "groups"
}

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w Group) PageTitle() string {
	return "Settings"
}

// object returns the model object being built. Implements the Builder interface.
func (w Group) object() data.Object {
	return w._group
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w Group) objectID() primitive.ObjectID {
	return w._group.GroupID
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w Group) objectType() string {
	return "Group"
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w Group) schema() schema.Schema {
	return schema.New(model.GroupSchema())
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w Group) service() service.ModelService {
	return w._factory.Group()
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
func (w Group) clone(action string) (Builder, error) {
	return NewGroup(w._factory, w._session, w._request, w._response, w._template, w._group, action)
}

/******************************************
 * Data Accessors
 ******************************************/

// IsAdminBuilder returns TRUE because Group is an admin route.
func (w Group) IsAdminBuilder() bool {
	return true
}

// GroupID returns the unique ID of the Group being built, as a string
func (w Group) GroupID() string {
	if w._group == nil {
		return ""
	}
	return w._group.GroupID.Hex()
}

// Label returns the label of the Group being built
func (w Group) Label() string {
	if w._group == nil {
		return ""
	}
	return w._group.Label
}

// Description returns the description of the Group being built
func (w Group) Description() string {
	if w._group == nil {
		return ""
	}
	return w._group.Description
}

// Icon returns the icon of the Group being built
func (w Group) Icon() string {
	return w._group.Icon
}

// IconWithDefault returns the icon, falling back to a default of the Group being built
func (w Group) IconWithDefault() string {
	return w._group.IconWithDefault()
}

/******************************************
 * QUERY BUILDERS
 ******************************************/

// Groups returns a QueryBuilder that lists Groups
func (w Group) Groups() *QueryBuilder[model.Group] {

	query := builder.NewBuilder().
		String("label").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.Group](w._factory.Group(), w._session, criteria)

	return &result
}

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w Group) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_group")
}
