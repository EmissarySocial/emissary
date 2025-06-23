package build

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Identity builds objects from any model service that implements the ModelService interface
type Identity struct {
	_identity *model.Identity
	CommonWithTemplate
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// NewIdentity returns a fully initialized `Identity` builder.
func NewIdentity(factory Factory, request *http.Request, response http.ResponseWriter, identity *model.Identity, actionID string) (Identity, error) {

	const location = "build.NewIdentity"

	// Get the `guest-profile` template.  This is the only template that works with the Identity builder.
	templateService := factory.Template()
	template, err := templateService.Load("guest")

	if err != nil {
		return Identity{}, derp.Wrap(err, location, "Cannot load template `guest`")
	}

	// RULE: The template must use the templateRole: "guest"
	if template.TemplateRole != "guest" {
		return Identity{}, derp.InternalError(location, "Identity template must use the TemplateRole `guest`", template.TemplateRole)
	}

	// RULE: The template must use the model: "identity"
	if template.Model != "Identity" {
		return Identity{}, derp.InternalError(location, "Identity template must use the Model `identity`", template.Model)
	}

	// Create a new CommonWithTemplate object, which will handle the common methods for this builder
	common, err := NewCommonWithTemplate(factory, request, response, template, identity, actionID)

	if err != nil {
		return Identity{}, derp.Wrap(err, "build.NewIdentity", "Error creating new model")
	}

	// Create the Identity builder
	builder := Identity{
		_identity:          identity,
		CommonWithTemplate: common,
	}

	// Done.
	return builder, nil
}

/******************************************
 * Custom Methods for Identity builder
 ******************************************/

// IdentityID returns the IdentityID property of this Identity
func (w Identity) IdentityID() primitive.ObjectID {
	return w._identity.IdentityID
}

// Name returns the Name property of this Identity
func (w Identity) Name() string {
	return w._identity.Name
}

// IconURL returns the IconURL property of this Identity
func (w Identity) IconURL() string {
	return w._identity.IconURL
}

// EmailAddress returns the EmailAddress property of this Identity
func (w Identity) EmailAddress() string {
	return w._identity.EmailAddress
}

// WebfingerHandle returns the WebfingerHandle (Fediverse username) property of this Identity
func (w Identity) WebfingerHandle() string {
	return w._identity.WebfingerHandle
}

// HasEmailAddress returns TRUE if this Identity has a non-zero email address
func (w Identity) HasEmailAddress() bool {
	return w._identity.HasEmailAddress()
}

// HasWebfingerHandle returns TRUE if this Identity has a non-zero webfinger handle
func (w Identity) HasWebfingerHandle() bool {
	return w._identity.HasWebfingerHandle()
}

// Icon returns an icon name to use for this Identity, based on the available identifiers.
func (w Identity) Icon() string {
	return w._identity.Icon()
}

// CreateDate returns the CreateDate property of this Identity
func (w Identity) CreateDate() int64 {
	return w._identity.CreateDate
}

// UpdateDate returns the UpdateDate property of this Identity
func (w Identity) UpdateDate() int64 {
	return w._identity.UpdateDate
}

func (w Identity) PrivilegeIDs() id.Slice {
	// Return the PrivilegeIDs property of this Identity
	return w._identity.PrivilegeIDs
}

// Privileges returns a QueryBuilder for the Privileges of the
// currently signed-in Identity
func (w Identity) Privileges() (QueryBuilder[model.Privilege], error) {

	// Define inbound parameters
	expressionBuilder := builder.NewBuilder().
		String("name")

	// Calculate criteria
	criteria := exp.And(
		exp.Equal("identityId", w._identity.IdentityID),
		expressionBuilder.Evaluate(w._request.URL.Query()),
	)

	// Return the query builder
	return NewQueryBuilder[model.Privilege](w._factory.Privilege(), criteria), nil
}

// PrivilegedStreams returns a map of the Streams that the
// currently signed-in Identity has privileges for
func (w Identity) PrivilegedStreams(privileges sliceof.Object[model.Privilege]) (mapof.Slices[primitive.ObjectID, primitive.ObjectID], error) {
	return w._factory.Stream().MapByPrivileges(privileges...)
}

/******************************************
 * Builder Interface
 ******************************************/

func (w Identity) object() data.Object {
	return w._identity
}

func (w Identity) objectType() string {
	return "Identity"
}

func (w Identity) objectID() primitive.ObjectID {
	return w._identity.IdentityID
}

func (w Identity) schema() schema.Schema {
	return schema.New(model.IdentitySchema())
}

func (w Identity) service() service.ModelService {
	return w._factory.Identity()
}

func (w Identity) Label() string {
	return w._identity.Name
}

func (w Identity) Token() string {
	return ""
}

func (w Identity) PageTitle() string {
	return w._identity.Name
}

func (w Identity) Permalink() string {
	return ""
}

func (w Identity) BasePath() string {
	return ""
}

func (w Identity) UserCan(string) bool {
	return false
}

func (w Identity) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Identity.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Stream
func (w Identity) View(actionID string) (template.HTML, error) {

	const location = "build.Identity.View"

	// Create a new builder (this will also validate the user's permissions)
	subStream, err := NewModel(w._factory, w._request, w._response, w._template, w._identity, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating sub-builder")
	}

	// Generate HTML template
	return subStream.Render()
}

func (w Identity) setState(stateID string) error {
	return nil
}

func (w Identity) clone(action string) (Builder, error) {
	return NewIdentity(w._factory, w._request, w._response, w._identity, action)
}

func (w Identity) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Model")
}
