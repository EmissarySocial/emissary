package render

import (
	"bytes"
	"html/template"
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message renders individual messages from a User's Inbox.
type Message struct {
	_service         *service.Inbox
	_message         *model.Message
	_activityStreams *service.ActivityStreams
	Common
}

// NewMessage returns a fully initialized `Message` renderer.
func NewMessage(factory Factory, request *http.Request, response http.ResponseWriter, modelService *service.Inbox, activityStreamsService *service.ActivityStreams, message *model.Message, actionID string) (Message, error) {

	const location = "render.NewMessage"

	// Load the Template
	templateService := factory.Template()
	template, err := templateService.Load("user-message") // TODO: Users should get to select their inbox template

	if err != nil {
		return Message{}, derp.Wrap(err, "render.NewInbox", "Error loading template")
	}

	// Validate the action
	action, ok := template.Action(actionID)

	if !ok {
		return Message{}, derp.NewBadRequestError(location, "Invalid action", actionID, template.Actions.Keys())
	}

	// Create the underlying Common renderer
	common, err := NewCommon(factory, request, response, template, actionID)

	if err != nil {
		return Message{}, derp.Wrap(err, location, "Error creating common renderer")
	}

	// Check permissions on the InboxFolder
	if !action.UserCan(message, &common._authorization) {
		if common._authorization.IsAuthenticated() {
			return Message{}, derp.NewForbiddenError(location, "Forbidden")
		} else {
			return Message{}, derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action", actionID)
		}
	}
	return Message{
		_service:         modelService,
		_message:         message,
		_activityStreams: activityStreamsService,
		Common:           common,
	}, nil
}

func (w Message) object() data.Object {
	return w._message
}

func (w Message) objectType() string {
	return w._service.ObjectType()
}

func (w Message) objectID() primitive.ObjectID {
	return w._message.MessageID
}

func (w Message) service() service.ModelService {
	return w._service
}

func (w Message) schema() schema.Schema {
	return w._service.Schema()
}

func (w Message) ObjectID() string {
	return w._message.MessageID.Hex()
}

func (w Message) Token() string {
	return ""
}

func (w Message) PageTitle() string {
	return ""
}

func (w Message) Permalink() string {
	return w._message.URL
}

func (w Message) UserCan(string) bool {
	return w._message.UserID == w.AuthenticatedID()
}

func (w Message) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w.action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "render.Message.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Stream
func (w Message) View(actionID string) (template.HTML, error) {

	const location = "render.Message.View"

	// Create a new renderer (this will also validate the user's permissions)
	subStream, err := NewMessage(w._factory, w._request, w._response, w._service, w._activityStreams, w._message, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating sub-renderer")
	}

	// Generate HTML template
	return subStream.Render()
}

func (w Message) templateRole() string {
	return "inbox"
}

func (w Message) clone(action string) (Renderer, error) {
	return NewMessage(w._factory, w._request, w._response, w._service, w._activityStreams, w._message, action)
}

/******************************************
 * Data Access Methods
 ******************************************/

// MessageID returns the inbox message ID for the object
func (w Message) MessageID() string {
	return w._message.MessageID.Hex()
}

// URL returns the public URL for the object
func (w Message) URL() string {
	return w._message.URL
}

// ActivityStream returns a hannibal Document that this message wraps
func (w Message) ActivityStream() streams.Document {
	result, err := w._activityStreams.Load(w._message.URL)
	derp.Report(err)
	return result
}

func (w Message) AttributedTo() model.PersonLink {
	return w._message.AttributedTo
}

func (w Message) InReplyTo() streams.Document {
	result, err := w._factory.ActivityStreams().Load(w._message.InReplyTo)
	derp.Report(err)
	return result
}

func (w Message) Label() string {
	return w._message.Label
}

func (w Message) HasSummary() bool {
	return w._message.HasSummary()
}

func (w Message) Summary() string {
	return w._message.Summary
}

func (w Message) SummaryOrContent() template.HTML {
	return template.HTML(w._message.SummaryOrContent())
}

func (w Message) ContentOrSummary() template.HTML {
	return template.HTML(w._message.ContentOrSummary())
}

func (w Message) HasImage() bool {
	return w._message.HasImage()
}

func (w Message) ImageURL() string {
	return w._message.ImageURL
}

func (w Message) HasContent() bool {
	return w._message.HasContent()
}

func (w Message) HasContentImage() bool {
	return w._message.HasContentImage()
}

func (w Message) ContentHTML() template.HTML {
	return template.HTML(w._message.ContentHTML)
}

func (w Message) FolderID() string {
	return w._message.FolderID.Hex()
}

func (w Message) Origin() model.OriginLink {
	return w._message.Origin
}

func (w Message) Rank() int64 {
	return w._message.Rank
}

func (w Message) PublishDate() int64 {
	return w._message.PublishDate
}

func (w Message) RepliesBefore(dateString string, maxRows int) sliceof.Object[streams.Document] {

	activityStreamsService := w._factory.ActivityStreams()
	maxDate := convert.Int64Default(dateString, math.MaxInt)
	result, _ := activityStreamsService.QueryRepliesBeforeDate(w._message.URL, maxDate, maxRows)

	return result.SliceOfDocuments()
}

func (w Message) RepliesAfter(dateString string, maxRows int) sliceof.Object[streams.Document] {
	minDate := convert.Int64(dateString)

	activityStreamsService := w._factory.ActivityStreams()
	result, _ := activityStreamsService.QueryRepliesAfterDate(w._message.URL, minDate, maxRows)

	return result.SliceOfDocuments()
}

// Responses generates a "Responses" renderer and passes it to the (hard-coded named) "responses" template.
// A default file is provided in the "base-social" template but can be overridden by other installed packages.
func (w Message) Responses() template.HTML {

	var buffer bytes.Buffer
	renderer := w.ResponsesRenderer()

	// Execute the "responses" template
	if err := w._template.HTMLTemplate.ExecuteTemplate(&buffer, "responses", renderer); err != nil {
		derp.Report(derp.Wrap(err, "render.Inbox.Responses", "Error rendering responses"))
	}

	// Celebrate with Triumph.
	return template.HTML(buffer.String())
}

// ResponsesRenderer returns a renderer for the responses widget.
func (w Message) ResponsesRenderer() Responses {

	// Collect values for Responses renderer
	userID := w.authorization().UserID
	internalURL := "/@me/messages/" + w._message.MessageID.Hex()
	responseService := w.factory().Response()

	// Create the new Responses renderer
	return NewResponses(userID, internalURL, w._message.URL, responseService)
}

func (w Message) debug() {
	log.Debug().Interface("object", w.object()).Msg("renderer_Message")
}
