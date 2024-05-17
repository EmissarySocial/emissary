package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.NewInternalError(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	followerService := factory.Follower()
	token := builder.QueryParam("followerId")
	follower := model.NewFollower()
	follower.ParentID = builder.AuthenticatedID() // TODO: Make this generic enough to work with Content Actors...

	// Only authenticated users can create new Follower records
	if (token == "") || (token == "new") {

		if !builder.IsAuthenticated() {
			return Halt().WithError(derp.NewForbiddenError(location, "Anonymous user is not authorized to perform this action"))
		}

	} else {

		// Existing Follower records can be looked up if authenticated, or with a secret.
		secret := builder.QueryParam("secret")
		if err := step.load(followerService, token, builder.AuthenticatedID(), secret, &follower); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Unable to load Follower via ID", token))
		}
	}

	// Create a new builder tied to the Follower record
	subBuilder, err := NewModel(factory, builder.request(), builder.response(), template, &follower, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the build pipeline on the Follower record
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

func (step StepWithFollower) load(followerService *service.Follower, token string, authenticatedID primitive.ObjectID, secret string, follower *model.Follower) error {

	const location = "build.StepWithFollower.load"

	// If a secret is present, then use it to load the Follower record
	if secret != "" {

		followerID, err := primitive.ObjectIDFromHex(token)

		if err != nil {
			return derp.Wrap(err, location, "Invalid follower ID", token)
		}

		return followerService.LoadBySecret(followerID, secret, follower)
	}

	// If the user is Authenticated, then use their Authentication to load the Follower record.
	if !authenticatedID.IsZero() {
		return followerService.LoadByToken(authenticatedID, token, follower)
	}

	// Nope.  Not happening.
	return derp.NewForbiddenError(location, "Anonymous user is not authorized to perform this action")

}
