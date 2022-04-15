package render

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepAddOutboxItem represents an action-step that can delete a Stream from the Domain
type StepAddOutboxItem struct {
	Type    string
	Link    string
	Comment *template.Template
}

// Get displays a customizable confirmation form for the delete
func (step StepAddOutboxItem) Get(renderer Renderer, buffer io.Writer) error {
	return step.addOutboxItem(renderer)
}

func (step StepAddOutboxItem) UseGlobalWrapper() bool {
	return true
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepAddOutboxItem) Post(renderer Renderer) error {
	return step.addOutboxItem(renderer)
}

func (step StepAddOutboxItem) addOutboxItem(renderer Renderer) error {

	const location = "render.StepAddOutboxItem.addOutboxItem"

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

	// Create the new outbox item
	streamService := factory.Stream()
	stream := model.NewStream()
	stream.TemplateID = "social-outbox-item"
	stream.ParentID = user.OutboxID
	stream.ParentIDs = []primitive.ObjectID{user.OutboxID}
	stream.Label = execTemplate(step.Comment, renderer)
	stream.SetAuthor(&user)

	stream.Data["type"] = step.Type

	switch step.Link {
	case "parent":
		stream.Data["link"] = renderer.Host() + "/" + stream.ParentID.Hex()

	// case "self":
	default:
		stream.Data["link"] = renderer.Host() + "/" + stream.StreamID.Hex()
	}

	if err := streamService.Save(&stream, ""); err != nil {
		return derp.Wrap(err, location, "Error writing outbox-item", stream)
	}

	return nil
}
