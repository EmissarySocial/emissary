package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithFollowing represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepWithFollowing struct {
	SubSteps []step.Step
}

func (step StepWithFollowing) Get(renderer Renderer, buffer io.Writer) ExitCondition {
	return step.execute(renderer, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithFollowing) Post(renderer Renderer, buffer io.Writer) ExitCondition {
	return step.execute(renderer, buffer, ActionMethodPost)
}

func (step StepWithFollowing) execute(renderer Renderer, buffer io.Writer, actionMethod ActionMethod) ExitCondition {

	const location = "render.StepWithFollowing.execute"

	if !renderer.IsAuthenticated() {
		return ExitError(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Collect required services and values
	factory := renderer.factory()
	context := renderer.context()
	followingService := factory.Following()
	followingToken := context.QueryParam("followingId")
	following := model.NewFollowing()
	following.UserID = renderer.AuthenticatedID()

	// If we have a real ID, then try to load the following from the database
	if (followingToken != "") && (followingToken != "new") {
		if err := followingService.LoadByToken(renderer.AuthenticatedID(), followingToken, &following); err != nil {
			if actionMethod == ActionMethodGet {
				return ExitError(derp.Wrap(err, location, "Unable to load Following", followingToken))
			}
			// Fall through for POSTS..  we're just creating a new following.
		}
	}

	// Create a new renderer tied to the Following record
	subRenderer, err := NewModel(factory, context, followingService, &following, renderer.template(), renderer.ActionID())

	if err != nil {
		return ExitError(derp.Wrap(err, location, "Unable to create sub-renderer"))
	}

	// Execute the POST render pipeline on the child
	status := Pipeline(step.SubSteps).Execute(factory, subRenderer, buffer, actionMethod)
	status.Error = derp.Wrap(status.Error, location, "Error executing steps for child")

	return ExitWithStatus(status)
}
