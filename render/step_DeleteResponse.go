package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepDeleteResponse represents an action-step that can delete a Stream from the Domain
type StepDeleteResponse struct{}

// Get displays a customizable confirmation form for the delete
func (step StepDeleteResponse) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepDeleteResponse) UseGlobalWrapper() bool {
	return true
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepDeleteResponse) Post(renderer Renderer, _ io.Writer) error {

	const location = "render.StepDeleteResponse.Post"

	// Collect URL parameters
	responseIDString := renderer.context().QueryParam("responseId")
	responseID, err := primitive.ObjectIDFromHex(responseIDString)

	if err != nil {
		return derp.Wrap(err, location, "Invalid response ID", responseIDString)
	}

	// Try to load the existing record
	responseService := renderer.factory().Response()
	response := model.NewResponse()

	if err := responseService.LoadByID(renderer.AuthenticatedID(), responseID, &response); err != nil {
		return derp.Wrap(err, location, "Error loading response", responseID)
	}

	// Try to delete the existing record
	if err := responseService.Delete(&response, "Deleted by User"); err != nil {
		return derp.Wrap(err, location, "Error deleting response", responseID)
	}

	return nil
}
