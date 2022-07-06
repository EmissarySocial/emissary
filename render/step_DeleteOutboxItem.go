package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
)

// StepDeleteOutboxItem represents an action-step that can delete an activity log item from a user's outbox.
type StepDeleteOutboxItem struct{}

// Get displays a customizable confirmation form for the delete
func (step StepDeleteOutboxItem) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepDeleteOutboxItem) UseGlobalWrapper() bool {
	return true
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepDeleteOutboxItem) Post(renderer Renderer) error {

	const location = "render.StepDeleteOutboxItem.Post"

	// Get the User's authorization for this request.
	authorization := renderer.authorization()

	// If the user is not signed in, then exit without error
	if !authorization.IsAuthenticated() {
		return nil
	}

	// Try to load the current user's record
	factory := renderer.factory()
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByID(authorization.UserID, &user); err != nil {
		return derp.Wrap(err, location, "Error loading current user from database", authorization.UserID)
	}

	// Try to find the original activity log
	streamService := factory.Stream()
	stream := model.NewStream()

	criteria := exp.Equal("parentId", user.OutboxID).
		AndEqual("data.originalStreamId", renderer.objectID())

	if err := streamService.Load(criteria, &stream); err != nil {
		// If it doesn't exist, then there's nothing to do.
		if derp.NotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Error locating original stream")
	}

	// When found, delete the outbox item
	if err := streamService.Delete(&stream, "Removed outbox item"); err != nil {
		return derp.Wrap(err, location, "Error deleting outbox item")
	}

	return nil
}
