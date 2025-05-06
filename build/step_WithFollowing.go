package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithFollowing is a Step that can update the data.DataMap custom data stored in a Stream
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

	// RULE: User MUST be authenticated to use this step
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.UnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	var userID primitive.ObjectID
	switch builder.IsAdminBuilder() {

	// Admin routes use zeroes for "Following" parents
	case true:

		// Must be an owner to use the admin route
		if !builder.IsOwner() {
			return Halt().WithError(derp.ForbiddenError(location, "User must be an owner to complete this action"))
		}

		userID = primitive.NilObjectID

	// Non-admin routes use the authenticated user's ID
	case false:
		userID = builder.AuthenticatedID()
	}

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.InternalError(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	followingService := factory.Following()
	token := builder.QueryParam("followingId")
	following := model.NewFollowing()
	following.UserID = userID

	// If we have a real ID, then try to load the following from the database
	if (token != "") && (token != "new") {
		if err := followingService.LoadByToken(userID, token, &following); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Following", token))
			}
			// Fall through for POSTS..  we're just creating a new following.
		}
	}

	// Create a new builder tied to the Following record
	subBuilder, err := NewModel(factory, builder.request(), builder.response(), template, &following, builder.actionID())

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
