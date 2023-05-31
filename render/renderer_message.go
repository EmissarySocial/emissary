package render

import (
	"bytes"
	"html/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	_service *service.Inbox
	_message *model.Message
	Common
}

func NewMessage(factory Factory, ctx *steranko.Context, modelService *service.Inbox, message *model.Message, template model.Template, actionID string) (Message, error) {

	const location = "render.NewMessage"

	action, ok := template.Action(actionID)

	if !ok {
		return Message{}, derp.NewBadRequestError(location, "Invalid action", actionID)
	}

	// Verify user's authorization to perform this Action on this Stream
	authorization := getAuthorization(ctx)

	// Check permissions on the InboxFolder
	if !action.UserCan(message, &authorization) {
		if authorization.IsAuthenticated() {
			return Message{}, derp.NewForbiddenError(location, "Forbidden")
		} else {
			return Message{}, derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action", actionID)
		}
	}

	// Create the underlying Common renderer
	common, err := NewCommon(factory, ctx, template, actionID)

	if err != nil {
		return Message{}, derp.Wrap(err, location, "Error creating common renderer")
	}

	return Message{
		_service: modelService,
		_message: message,
		Common:   common,
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
	return ""
}

func (w Message) UserCan(string) bool {
	return false
}

func (w Message) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Stream.Render", "Error generating HTML"))
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Stream
func (w Message) View(actionID string) (template.HTML, error) {

	const location = "render.Stream.View"

	// Create a new renderer (this will also validate the user's permissions)
	subStream, err := NewMessage(w._factory, w._context, w._service, w._message, w._template, actionID)

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
	return NewMessage(w._factory, w._context, w._service, w._message, w._template, action)
}

/******************************************
 * Data Access Methods
 ******************************************/

func (w Message) URL() string {
	return w._message.URL
}

func (w Message) Label() string {
	return w._message.Label
}

func (w Message) Summary() string {
	return w._message.Summary
}

func (w Message) SummaryHTML() template.HTML {
	return template.HTML(w._message.Summary)
}

func (w Message) ImageURL() string {
	return w._message.ImageURL
}

func (w Message) AttributedTo() sliceof.Object[model.PersonLink] {
	return w._message.AttributedTo
}

func (w Message) InReplyTo() streams.Document {
	result, _ := w._factory.ActivityStreams().Load(w._message.InReplyTo)
	return result
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
