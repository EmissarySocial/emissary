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
func NewIdentity(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, identity *model.Identity, actionID string) (Identity, error) {

	const location = "build.NewIdentity"

	// Get the `guest-profile` template.  This is the only template that works with the Identity builder.
	templateService := factory.Template()
	template, err := templateService.Load("guest")

	if err != nil {
		return Identity{}, derp.Wrap(err, location, "Cannot load template `guest`")
	}

	// RULE: The template must use the templateRole: "guest"
	if template.TemplateRole != "guest" {
		return Identity{}, derp.Internal(location, "Identity template must use the TemplateRole `guest`", template.TemplateRole)
	}

	// RULE: The template must use the model: "identity"
	if template.Model != "Identity" {
		return Identity{}, derp.Internal(location, "Identity template must use the Model `identity`", template.Model)
	}

	// Create a new CommonWithTemplate object, which will handle the common methods for this builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, identity, actionID)

	if err != nil {
		return Identity{}, derp.Wrap(err, "build.NewIdentity", "Creating new model")
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

// HasEmailAddress returns TRUE if this Identity has a non-zero email address
func (w Identity) HasEmailAddress() bool {
	return w._identity.HasEmailAddress()
}

// HasWebfingerUsername returns TRUE if this Identity has a non-zero webfinger handle
func (w Identity) HasWebfingerUsername() bool {
	return w._identity.HasWebfingerUsername()
}

// HasActivityPubActor returns TRUE if this Identity has a non-zero webfinger handle
func (w Identity) HasActivityPubActor() bool {
	return w._identity.HasActivityPubActor()
}

// EmailAddress returns the EmailAddress property of this Identity
func (w Identity) EmailAddress() string {
	return w._identity.EmailAddress
}

// ActivityPubActor returns the ActivityPubActor property of this Identity
func (w Identity) ActivityPubActor() string {
	return w._identity.ActivityPubActor
}

// WebfingerUsername returns the WebfingerUsername property of this Identity
func (w Identity) WebfingerUsername() string {
	return w._identity.WebfingerUsername
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

// PrivilegeIDs returns the privilege IDs of the Identity being built
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
		exp.Equal("isVisible", true),
		expressionBuilder.Evaluate(w._request.URL.Query()),
	)

	// Return the query builder
	return NewQueryBuilder[model.Privilege](w._factory.Privilege(), w._session, criteria), nil
}

// PrivilegedStreams returns a map of the Streams that the
// currently signed-in Identity has privileges for
func (w Identity) PrivilegedStreams(privileges sliceof.Object[model.Privilege]) (mapof.Slices[primitive.ObjectID, primitive.ObjectID], error) {
	return w._factory.Stream().MapByPrivileges(w._session, privileges...)
}

/******************************************
 * Builder Interface
 ******************************************/

// object returns the model object being built. Implements the Builder interface.
func (w Identity) object() data.Object {
	return w._identity
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w Identity) objectType() string {
	return "Identity"
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w Identity) objectID() primitive.ObjectID {
	return w._identity.IdentityID
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w Identity) schema() schema.Schema {
	return schema.New(model.IdentitySchema())
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w Identity) service() service.ModelService {
	return w._factory.Identity()
}

// Label returns the label of the Identity being built
func (w Identity) Label() string {
	return w._identity.Name
}

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w Identity) Token() string {
	return ""
}

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w Identity) PageTitle() string {
	return w._identity.Name
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w Identity) Permalink() string {
	return ""
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w Identity) BasePath() string {
	return ""
}

// UserCan returns TRUE if the signed-in User may run the named action. Implements the Builder interface.
func (w Identity) UserCan(string) bool {
	return false
}

// Render builds this object into HTML. Implements the Builder interface.
func (w Identity) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Identity.Render", "Generating HTML")
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
	subStream, err := NewModel(w._factory, w._session, w._request, w._response, w._template, w._identity, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Creating sub-builder")
	}

	// Generate HTML template
	return subStream.Render()
}

// setState moves the underlying object into the provided state
func (w Identity) setState(stateID string) error {
	return nil
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
func (w Identity) clone(action string) (Builder, error) {
	return NewIdentity(w._factory, w._session, w._request, w._response, w._identity, action)
}

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w Identity) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Model")
}
