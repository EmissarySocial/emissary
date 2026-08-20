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
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchTag is a builder for the admin/searchTags page
// It can only be accessed by a Domain Owner
type SearchTag struct {
	_searchTag *model.SearchTag
	CommonWithTemplate
}

// NewSearchTag returns a fully initialized `SearchTag` builder.
func NewSearchTag(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, template model.Template, searchTag *model.SearchTag, actionID string) (SearchTag, error) {

	const location = "build.NewSearchTag"

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, searchTag, actionID)

	if err != nil {
		return SearchTag{}, derp.Wrap(err, location, "Creating common builder")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return SearchTag{}, derp.Forbidden(location, "Must be domain owner to continue")
	}

	// Return the SearchTag builder
	return SearchTag{
		_searchTag:         searchTag,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Stream
func (w SearchTag) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.SearchTag.Render", "Generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this SearchTag
func (w SearchTag) View(actionID string) (template.HTML, error) {

	const location = "build.SearchTag.View"

	builder, err := NewSearchTag(w._factory, w._session, w._request, w._response, w._template, w._searchTag, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Creating SearchTag builder")
	}

	return builder.Render()
}

// NavigationID returns the top-level navigation item to highlight. Implements the Builder interface.
func (w SearchTag) NavigationID() string {
	return "admin"
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w SearchTag) Permalink() string {
	return w.Host() + "/admin/searchTags/" + w.SearchTagID()
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w SearchTag) BasePath() string {
	return "/admin/searchTags/" + w.SearchTagID()
}

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w SearchTag) Token() string {
	return "tags"
}

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w SearchTag) PageTitle() string {
	return "Settings"
}

// object returns the model object being built. Implements the Builder interface.
func (w SearchTag) object() data.Object {
	return w._searchTag
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w SearchTag) objectID() primitive.ObjectID {
	return w._searchTag.SearchTagID
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w SearchTag) objectType() string {
	return "SearchTag"
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w SearchTag) schema() schema.Schema {
	return schema.New(model.SearchTagSchema())
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w SearchTag) service() service.ModelService {
	return w._factory.SearchTag()
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
func (w SearchTag) clone(action string) (Builder, error) {
	return NewSearchTag(w._factory, w._session, w._request, w._response, w._template, w._searchTag, action)
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IsAdminBuilder returns TRUE because SearchTag is an admin route.
func (w SearchTag) IsAdminBuilder() bool {
	return true
}

// SearchTagID returns the unique ID of the SearchTag being built, as a string
func (w SearchTag) SearchTagID() string {
	if w._searchTag == nil {
		return ""
	}
	return w._searchTag.SearchTagID.Hex()
}

// Name returns the name of the SearchTag being built
func (w SearchTag) Name() string {
	if w._searchTag == nil {
		return ""
	}
	return w._searchTag.Name
}

/******************************************
 * Query Builders
 ******************************************/

// SearchTags returns a query builder for all SearchTags in the datatabase.
func (w SearchTag) SearchTags() *QueryBuilder[model.SearchTag] {

	query := builder.NewBuilder().
		String("name", builder.WithDefaultOpBeginsWith()).
		String("group").
		Int("stateId")

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.SearchTag](w._factory.SearchTag(), w._session, criteria)
	result.CaseInsensitive()

	return &result
}

// States returns the moderation states that a SearchTag can be placed in, as form lookup codes
func (w SearchTag) States() []form.LookupCode {
	return w.lookupProvider().Group("searchTag-states").Get()
}

// Groups returns the distinct group names in use across all SearchTags
func (w SearchTag) Groups() []form.LookupCode {
	return w._factory.SearchTag().ListGroups(w._session)
}

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w SearchTag) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_searchTag")
}
