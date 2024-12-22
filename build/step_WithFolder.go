package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithFolder is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithFolder struct {
	SubSteps []step.Step
}

func (step StepWithFolder) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithFolder) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithFolder) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithFolder.execute"

	// RULE: User MUST be authenticated to use this step
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.NewInternalError(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	folderService := factory.Folder()
	folderToken := builder.QueryParam("folderId")
	folder := model.NewFolder()
	folder.UserID = builder.AuthenticatedID()

	// If we have a real ID, then try to load the folder from the database
	if (folderToken != "") && (folderToken != "new") {
		if err := folderService.LoadByToken(builder.AuthenticatedID(), folderToken, &folder); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Folder", folderToken))
			}
			// Fall through for POSTS..  we're just creating a new folder.
		}
	}

	// Create a new builder tied to the Folder record
	subBuilder, err := NewModel(factory, builder.request(), builder.response(), template, &folder, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}
