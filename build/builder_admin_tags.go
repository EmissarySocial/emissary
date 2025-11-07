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
		return SearchTag{}, derp.Wrap(err, location, "Unable to create common builder")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return SearchTag{}, derp.ForbiddenError(location, "Must be domain owner to continue")
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
		err := derp.Wrap(status.Error, "build.SearchTag.Render", "Error generating HTML")
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
		return template.HTML(""), derp.Wrap(err, location, "Unable to create SearchTag builder")
	}

	return builder.Render()
}

func (w SearchTag) NavigationID() string {
	return "admin"
}

func (w SearchTag) Permalink() string {
	return w.Hostname() + "/admin/searchTags/" + w.SearchTagID()
}

func (w SearchTag) BasePath() string {
	return "/admin/searchTags/" + w.SearchTagID()
}

func (w SearchTag) Token() string {
	return "tags"
}

func (w SearchTag) PageTitle() string {
	return "Settings"
}

func (w SearchTag) object() data.Object {
	return w._searchTag
}

func (w SearchTag) objectID() primitive.ObjectID {
	return w._searchTag.SearchTagID
}

func (w SearchTag) objectType() string {
	return "SearchTag"
}

func (w SearchTag) schema() schema.Schema {
	return schema.New(model.SearchTagSchema())
}

func (w SearchTag) service() service.ModelService {
	return w._factory.SearchTag()
}

func (w SearchTag) clone(action string) (Builder, error) {
	return NewSearchTag(w._factory, w._session, w._request, w._response, w._template, w._searchTag, action)
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IsAdminBuilder returns TRUE because SearchTag is an admin route.
func (w SearchTag) IsAdminBuilder() bool {
	return false
}

func (w SearchTag) SearchTagID() string {
	if w._searchTag == nil {
		return ""
	}
	return w._searchTag.SearchTagID.Hex()
}

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

func (w SearchTag) States() []form.LookupCode {
	return w.lookupProvider().Group("searchTag-states").Get()
}

func (w SearchTag) Groups() []form.LookupCode {
	return w._factory.SearchTag().ListGroups(w._session)
}

func (w SearchTag) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_searchTag")
}
