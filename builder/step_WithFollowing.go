package builder

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

func (step StepWithFollowing) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithFollowing) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithFollowing) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithFollowing.execute"

	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Collect required services and values
	factory := builder.factory()
	followingService := factory.Following()
	token := builder.QueryParam("followingId")
	following := model.NewFollowing()
	following.UserID = builder.AuthenticatedID()

	// If we have a real ID, then try to load the following from the database
	if (token != "") && (token != "new") {
		if err := followingService.LoadByToken(builder.AuthenticatedID(), token, &following); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Following", token))
			}
			// Fall through for POSTS..  we're just creating a new following.
		}
	}

	// Create a new builder tied to the Following record
	subBuilder, err := NewModel(factory, builder.request(), builder.response(), &following, builder.template(), builder.ActionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)

	if result.Error != nil {
		return Halt().WithError(derp.Wrap(result.Error, location, "Error executing steps for child"))
	}

	return UseResult(result)
}
