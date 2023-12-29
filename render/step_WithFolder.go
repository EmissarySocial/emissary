package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithFolder represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithFolder struct {
	SubSteps []step.Step
}

func (step StepWithFolder) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithFolder) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodPost)
}

func (step StepWithFolder) execute(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "render.StepWithFolder.execute"

	if !renderer.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Collect required services and values
	factory := renderer.factory()
	folderService := factory.Folder()
	folderToken := renderer.QueryParam("folderId")
	folder := model.NewFolder()
	folder.UserID = renderer.AuthenticatedID()

	// If we have a real ID, then try to load the folder from the database
	if (folderToken != "") && (folderToken != "new") {
		if err := folderService.LoadByToken(renderer.AuthenticatedID(), folderToken, &folder); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Folder", folderToken))
			}
			// Fall through for POSTS..  we're just creating a new folder.
		}
	}

	// Create a new renderer tied to the Folder record
	subRenderer, err := NewModel(factory, renderer.request(), renderer.response(), &folder, renderer.template(), renderer.ActionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-renderer"))
	}

	// Execute the POST render pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subRenderer, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}
