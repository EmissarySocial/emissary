package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithBlock represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithBlock struct {
	SubSteps []step.Step
}

func (step StepWithBlock) Get(renderer Renderer, buffer io.Writer) error {
	return step.doStep(renderer, buffer, ActionMethodGet)
}

func (step StepWithBlock) UseGlobalWrapper() bool {
	return useGlobalWrapper(step.SubSteps)
}

// Post updates the stream with approved data from the request body.
func (step StepWithBlock) Post(renderer Renderer) error {
	return step.doStep(renderer, nil, ActionMethodPost)
}

func (step StepWithBlock) doStep(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) error {

	const location = "render.StepWithBlock.doStep"

	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action")
	}

	// Collect required services and values
	factory := renderer.factory()
	context := renderer.context()
	blockService := factory.Block()
	blockToken := context.QueryParam("blockId")
	block := model.NewBlock()
	block.UserID = renderer.AuthenticatedID()

	// If we have a real ID, then try to load the block from the database
	if blockToken != "new" {
		if err := blockService.LoadByToken(renderer.AuthenticatedID(), blockToken, &block); err != nil {
			if actionMethod == ActionMethodGet {
				return derp.Wrap(err, location, "Unable to load Block", blockToken)
			}
			// Fall through for POSTS..  we're just creating a new block.
		}
	}

	// Create a new renderer tied to the Block record
	subRenderer, err := NewModel(factory, context, blockService, &block, renderer.template(), renderer.ActionID())

	if err != nil {
		return derp.Wrap(err, location, "Unable to create sub-renderer")
	}

	// Execute the POST render pipeline on the child
	if err := Pipeline(step.SubSteps).Execute(factory, subRenderer, buffer, actionMethod); err != nil {
		return derp.Wrap(err, location, "Error executing steps for child")
	}

	return nil
}
