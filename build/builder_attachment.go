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

// Attachment builds objects from any model service that implements the ModelService interface
type Attachment struct {
	_attachment *model.Attachment
	CommonWithTemplate
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// NewAttachment returns a fully initialized `Attachment` builder.
func NewAttachment(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, template model.Template, attachment *model.Attachment, actionID string) (Attachment, error) {

	common, err := NewCommonWithTemplate(factory, session, request, response, template, attachment, actionID)

	if err != nil {
		return Attachment{}, derp.Wrap(err, "build.NewAttachment", "Creating new model")
	}

	builder := Attachment{
		_attachment:        attachment,
		CommonWithTemplate: common,
	}

	return builder, nil
}

/******************************************
 * Custom Methods for Attachment builder
 ******************************************/

// AmFollowing returns a Following record for the current user and the given URL
// If the user is not authenticated, or the URL is not valid, then an empty Following record is returned.
// The UX uses this to label "mutual" follows
func (w Attachment) AmFollowing(url string) model.Following {

	if !w._authorization.IsAuthenticated() {
		return model.NewFollowing()
	}

	// Get following service and new following record
	followingService := w._factory.Following()
	following := model.NewFollowing()

	// Retrieve following record. Discard errors
	// nolint:errcheck
	_ = followingService.LoadByURL(w._session, w._authorization.UserID, url, &following)

	// Return the (possibly empty) Following record
	return following
}

// AttachmentID returns the AttachmentID property of this Follow
func (w Attachment) AttachmentID() string {
	return w._attachment.AttachmentID.Hex()
}

// ObjectType returns the ObjectType property of this Follow
func (w Attachment) ObjectType() string {
	return w._attachment.ObjectType
}

// ObjectID returns the ObjectID property of this Follow
func (w Attachment) ObjectID() string {
	return w._attachment.ObjectID.Hex()
}

// CreateDate returns the CreateDate property of this Follow
func (w Attachment) CreateDate() int64 {
	return w._attachment.CreateDate
}

// UpdateDate returns the UpdateDate property of this Follow
func (w Attachment) UpdateDate() int64 {
	return w._attachment.UpdateDate
}

// Original returns the original of the Attachment being built
func (w Attachment) Original() string {
	return w._attachment.Original
}

// Category returns the category of the Attachment being built
func (w Attachment) Category() string {
	return w._attachment.Category
}

// Label returns the label of the Attachment being built
func (w Attachment) Label() string {
	return w._attachment.Label
}

// Description returns the description of the Attachment being built
func (w Attachment) Description() string {
	return w._attachment.Description
}

// URL returns the URL of the Attachment being built
func (w Attachment) URL() string {
	return w._attachment.URL
}

// Status returns the status of the Attachment being built
func (w Attachment) Status() string {
	return w._attachment.Status
}

// Rules returns the rules of the Attachment being built
func (w Attachment) Rules() model.AttachmentRules {
	return w._attachment.Rules
}

// Height returns the height of the Attachment being built
func (w Attachment) Height() int {
	return w._attachment.Height
}

// Width returns the width of the Attachment being built
func (w Attachment) Width() int {
	return w._attachment.Width
}

// Duration returns the duration of the Attachment being built
func (w Attachment) Duration() int {
	return w._attachment.Duration
}

// Rank returns the rank of the Attachment being built
func (w Attachment) Rank() int {
	return w._attachment.Rank
}

/******************************************
 * Builder Interface
 ******************************************/

// object returns the model object being built. Implements the Builder interface.
func (w Attachment) object() data.Object {
	return w._attachment
}

// objectType returns the name of the model type being built. Implements the Builder interface.
func (w Attachment) objectType() string {
	return "Attachment"
}

// objectID returns the unique ID of the object being built. Implements the Builder interface.
func (w Attachment) objectID() primitive.ObjectID {
	return w._attachment.AttachmentID
}

// schema returns the rosetta schema that validates this object. Implements the Builder interface.
func (w Attachment) schema() schema.Schema {
	return schema.New(model.AttachmentSchema())
}

// service returns the ModelService that backs this Builder. Implements the Builder interface.
func (w Attachment) service() service.ModelService {
	return w._factory.Attachment()
}

// Token returns the URL token of the record being built. Implements the Builder interface.
func (w Attachment) Token() string {
	return w._attachment.AttachmentID.Hex()
}

// PageTitle returns the human-friendly title to display at the top of the page. Implements the Builder interface.
func (w Attachment) PageTitle() string {
	return w._attachment.Label
}

// Permalink returns the permanent URL of the record being built. Implements the Builder interface.
func (w Attachment) Permalink() string {
	return w._attachment.CalcURL(w._factory.Host())
}

// BasePath returns the URL path of this object, without any action. Implements the Builder interface.
func (w Attachment) BasePath() string {
	return w._attachment.CalcURL(w._factory.Host())
}

// UserCan returns TRUE if the signed-in User may run the named action. Implements the Builder interface.
func (w Attachment) UserCan(string) bool {
	return false
}

// Render builds this object into HTML. Implements the Builder interface.
func (w Attachment) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Attachment.Render", "Generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Stream
func (w Attachment) View(actionID string) (template.HTML, error) {

	const location = "build.Attachment.View"

	// Create a new builder (this will also validate the user's permissions)
	subStream, err := NewModel(w._factory, w._session, w._request, w._response, w._template, w._attachment, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Creating sub-builder")
	}

	// Generate HTML template
	return subStream.Render()
}

// setState moves the underlying object into the provided state
func (w Attachment) setState(stateID string) error {
	return nil
}

// clone returns a copy of this Builder, pointed at a different action. Implements the Builder interface.
func (w Attachment) clone(action string) (Builder, error) {
	return NewAttachment(w._factory, w._session, w._request, w._response, w._template, w._attachment, action)
}

// debug writes this Builder's state to the console. Implements the Builder interface.
func (w Attachment) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Model")
}
