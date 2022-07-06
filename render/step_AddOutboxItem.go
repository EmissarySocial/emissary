package render

import (
	"io"
	"text/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepAddOutboxItem represents an action-step that can add an activity log to a user's outbox
type StepAddOutboxItem struct {
	Label       *template.Template
	Description *template.Template
	Type        string
	Link        string
}

// Get displays a customizable confirmation form for the delete
func (step StepAddOutboxItem) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepAddOutboxItem) UseGlobalWrapper() bool {
	return true
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepAddOutboxItem) Post(renderer Renderer) error {

	const location = "render.StepAddOutboxItem.Post"

	// Get the User's authorization for this request.
	authorization := renderer.authorization()

	// If the user is not signed in, then exit without error
	if !authorization.IsAuthenticated() {
		return nil
	}

	// Verify that this system uses outbox items.
	factory := renderer.factory()
	templateService := factory.Template()
	if _, err := templateService.Load("social-outbox-item"); err != nil {
		// Non-standard error handling here.  If there is no outbox item template,
		// then exit without error.
		return nil
	}

	// Try to load the current user's record
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByID(authorization.UserID, &user); err != nil {
		return derp.Wrap(err, location, "Error loading current user from database", authorization.UserID)
	}

	// Now let's get to the stream...
	streamService := factory.Stream()

	// If there is already an outbox item connected to this stream, then remove it before creating a duplicate
	if err := streamService.DeleteRelatedDuplicate(user.OutboxID, renderer.objectID()); err != nil {
		return derp.Wrap(err, location, "Error deleting related duplicate")
	}

	// Create the new outbox item
	stream := model.NewStream()
	stream.TemplateID = "social-outbox-item"
	stream.ParentID = user.OutboxID
	stream.ParentIDs = []primitive.ObjectID{user.OutboxID}
	stream.Label = execTemplate(step.Label, renderer)
	stream.Description = execTemplate(step.Description, renderer)
	stream.SetAuthor(&user)
	stream.Data["type"] = step.Type
	stream.Data["originalStreamId"] = renderer.objectID()

	switch step.Link {
	case "parent":
		if streamRenderer, ok := renderer.(*Stream); ok {
			stream.Data["link"] = renderer.Host() + "/" + streamRenderer.stream.ParentID.Hex()
		}

	// case "self":
	default:
		stream.Data["link"] = renderer.Host() + "/" + renderer.objectID().Hex()
	}

	if err := streamService.Save(&stream, ""); err != nil {
		return derp.Wrap(err, location, "Error writing outbox-item", stream)
	}

	return nil
}
