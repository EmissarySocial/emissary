package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithFollower represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithFollower struct {
	SubSteps []step.Step
}

func (step StepWithFollower) Get(renderer Renderer, buffer io.Writer) error {
	return step.doStep(renderer, buffer, ActionMethodGet)
}

func (step StepWithFollower) UseGlobalWrapper() bool {
	return useGlobalWrapper(step.SubSteps)
}

// Post updates the stream with approved data from the request body.
func (step StepWithFollower) Post(renderer Renderer) error {
	return step.doStep(renderer, nil, ActionMethodPost)
}

func (step StepWithFollower) doStep(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) error {

	const location = "render.StepWithFollower.doStep"

	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action")
	}

	// Collect required services and values
	factory := renderer.factory()
	context := renderer.context()
	followerService := factory.Follower()
	followerToken := context.QueryParam("followerId")
	follower := model.NewFollower()
	follower.ParentID = renderer.AuthenticatedID()

	// If we have a real ID, then try to load the follower from the database
	if (followerToken != "") && (followerToken != "new") {
		if err := followerService.LoadByToken(renderer.AuthenticatedID(), followerToken, &follower); err != nil {
			if actionMethod == ActionMethodGet {
				return derp.Wrap(err, location, "Unable to load Follower", followerToken)
			}
			// Fall through for POSTS..  we're just creating a new follower.
		}
	}

	// Create a new renderer tied to the Follower record
	subRenderer, err := NewModel(factory, context, followerService, &follower, renderer.template(), renderer.ActionID())

	if err != nil {
		return derp.Wrap(err, location, "Unable to create sub-renderer")
	}

	// Execute the POST render pipeline on the child
	if err := Pipeline(step.SubSteps).Execute(factory, subRenderer, buffer, actionMethod); err != nil {
		return derp.Wrap(err, location, "Error executing steps for child")
	}

	return nil
}
