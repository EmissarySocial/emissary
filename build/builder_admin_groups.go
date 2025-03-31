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
func NewGroup(factory Factory, request *http.Request, response http.ResponseWriter, template model.Template, group *model.Group, actionID string) (Group, error) {

	const location = "build.NewGroup"

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, request, response, template, actionID)

	if err != nil {
		return Group{}, derp.Wrap(err, location, "Error creating common builder")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return Group{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
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
		err := derp.Wrap(status.Error, "build.Group.Render", "Error generating HTML")
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

	builder, err := NewGroup(w._factory, w._request, w._response, w._template, w._group, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Group builder")
	}

	return builder.Render()
}

func (w Group) NavigationID() string {
	return "admin"
}

func (w Group) Permalink() string {
	return w.Hostname() + "/groups/" + w.GroupID()
}

func (w Group) BasePath() string {
	if w._group == nil {
		return "/groups"
	}
	return "/groups/" + w.GroupID()
}

func (w Group) Token() string {
	return "groups"
}

func (w Group) PageTitle() string {
	return "Settings"
}

func (w Group) object() data.Object {
	return w._group
}

func (w Group) objectID() primitive.ObjectID {
	return w._group.GroupID
}

func (w Group) objectType() string {
	return "Group"
}

func (w Group) schema() schema.Schema {
	return schema.New(model.GroupSchema())
}

func (w Group) service() service.ModelService {
	return w._factory.Group()
}

func (w Group) clone(action string) (Builder, error) {
	return NewGroup(w._factory, w._request, w._response, w._template, w._group, action)
}

/******************************************
 * DATA ACCESSORS
 ******************************************/

// IsAdminBuilder returns TRUE because Group is an admin route.
func (w Group) IsAdminBuilder() bool {
	return false
}

func (w Group) GroupID() string {
	if w._group == nil {
		return ""
	}
	return w._group.GroupID.Hex()
}

func (w Group) Label() string {
	if w._group == nil {
		return ""
	}
	return w._group.Label
}

/******************************************
 * QUERY BUILDERS
 ******************************************/

func (w Group) Groups() *QueryBuilder[model.Group] {

	query := builder.NewBuilder().
		String("label").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.Group](w._factory.Group(), criteria)

	return &result
}

func (w Group) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_group")
}
