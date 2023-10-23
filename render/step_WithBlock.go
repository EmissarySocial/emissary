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

func (step StepWithBlock) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithBlock) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return step.execute(renderer, buffer, ActionMethodPost)
}

func (step StepWithBlock) execute(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "render.StepWithBlock.doStep"

	if !renderer.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Collect required services and values
	factory := renderer.factory()
	blockService := factory.Block()
	blockToken := renderer.QueryParam("blockId")
	block := model.NewBlock()
	block.UserID = renderer.AuthenticatedID()

	if (blockToken != "") && (blockToken != "new") {
		if err := blockService.LoadByToken(renderer.AuthenticatedID(), blockToken, &block); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Block", blockToken))
			}
			// Fall through for POSTS..  we're just creating a new block.
		}
	}

	// Create a new renderer tied to the Block record
	subRenderer, err := NewModel(factory, renderer.request(), renderer.response(), blockService, &block, renderer.template(), renderer.ActionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-renderer"))
	}

	// Execute the POST render pipeline on the child
	reesult := Pipeline(step.SubSteps).Execute(factory, subRenderer, buffer, actionMethod)
	reesult.Error = derp.Wrap(reesult.Error, location, "Error executing steps for child")

	return UseResult(reesult)
}
