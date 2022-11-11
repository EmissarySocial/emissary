package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithInboxFolder represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithInboxFolder struct {
	SubSteps []step.Step
}

func (step StepWithInboxFolder) Get(renderer Renderer, buffer io.Writer) error {
	return step.doStep(renderer, buffer, ActionMethodGet)
}

func (step StepWithInboxFolder) UseGlobalWrapper() bool {
	return useGlobalWrapper(step.SubSteps)
}

// Post updates the stream with approved data from the request body.
func (step StepWithInboxFolder) Post(renderer Renderer) error {
	return step.doStep(renderer, nil, ActionMethodPost)
}

func (step StepWithInboxFolder) doStep(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) error {

	const location = "render.StepWithInboxFolder.doStep"

	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action")
	}

	// Collect required services and values
	factory := renderer.factory()
	inboxFolderService := factory.InboxFolder()
	inboxFolderToken := renderer.context().QueryParam("inboxFolderId")
	inboxFolder := model.NewInboxFolder()

	// If we have a real ID, then try to load the folder from the database
	if inboxFolderToken != "new" {

		if err := inboxFolderService.LoadByToken(renderer.AuthenticatedID(), inboxFolderToken, &inboxFolder); err != nil {
			return derp.Wrap(err, location, "Unable to load InboxFolder", inboxFolderToken)
		}
	}

	// Create a new renderer tied to the InboxFolder record
	subRenderer, err := NewModel(factory, renderer.context(), inboxFolderService, &inboxFolder, renderer.template(), "view")

	if err != nil {
		return derp.Wrap(err, location, "Unable to create sub-renderer")
	}

	// Execute the POST render pipeline on the child
	if err := Pipeline(step.SubSteps).Execute(factory, subRenderer, buffer, actionMethod); err != nil {
		return derp.Wrap(err, location, "Error executing steps for child")
	}

	return nil
}

/*

const location = "render.StepWithInboxFolder.Post"

factory := renderer.factory()
streamRenderer := renderer.(*Stream)

children, err := factory.Stream().ListByParent(streamRenderer.stream.ParentID)

if err != nil {
	return derp.Wrap(err, location, "Error listing children")
}

child := model.NewStream()

for children.Next(&child) {

	// Make a renderer with the new child stream
	// TODO: Is "view" really the best action to use here??
	childStream, err := NewStreamWithoutTemplate(streamRenderer.factory(), streamRenderer.context(), &child, "")

	if err != nil {
		return derp.Wrap(err, location, "Error creating renderer for child")
	}

	// Execute the POST render pipeline on the child
	if err := Pipeline(step.SubSteps).Post(factory, &childStream); err != nil {
		return derp.Wrap(err, location, "Error executing steps for child")
	}

	// Reset the child object so that old records don't bleed into new ones.
	child = model.NewStream()
}
*/
