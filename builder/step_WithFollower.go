package builder

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

func (step StepWithFollower) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithFollower) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithFollower) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithFollower.execute"

	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Collect required services and values
	factory := builder.factory()
	followerService := factory.Follower()
	followerToken := builder.QueryParam("followerId")
	follower := model.NewFollower()
	follower.ParentID = builder.AuthenticatedID()

	// Try to load the Follower record (unless we're creating a NEW record)
	if (followerToken != "") && (followerToken != "new") {
		if err := followerService.LoadByToken(builder.AuthenticatedID(), followerToken, &follower); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Follower via ID", followerToken))
			}
		}
	}

	// Create a new builder tied to the Follower record
	subBuilder, err := NewModel(factory, builder.request(), builder.response(), &follower, builder.template(), builder.ActionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the build pipeline on the Follower record
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}
