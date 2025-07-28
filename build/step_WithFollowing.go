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

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.InternalError(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	following, err := step.getFollowing(builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to get Following record"))
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

func (step StepWithFollowing) getFollowing(builder Builder) (model.Following, error) {

	const location = "build.StepWithFollowing.getFollowing"

	// Locate the User ID for this request
	userID, err := step.getUserID(builder)

	if err != nil {
		return model.NewFollowing(), derp.Wrap(err, location, "Unable to locate User ID")
	}

	// Collect required services and values
	factory := builder.factory()
	followingService := factory.Following()
	following := model.NewFollowing()
	following.UserID = userID

	// If a `url` query parameter is provided, then use it to load the Following record
	if url := builder.QueryParam("url"); url != "" {

		if err := followingService.LoadByURL(userID, url, &following); !derp.IsNilOrNotFound(err) {
			return model.NewFollowing(), derp.Wrap(err, location, "Unable to load Following by URL", url)
		}

		following.URL = url
		return following, nil
	}

	// Otherwise, use the `followingId` query aprameter to load the Following record
	token := builder.QueryParam("followingId")

	// Finally, try to load the Following record from the database.
	if err := followingService.LoadByToken(userID, token, &following); err != nil {
		return following, derp.Wrap(err, location, "Unable to load Following", token)
	}

	// Success.
	return following, nil
}

func (step StepWithFollowing) getUserID(builder Builder) (primitive.ObjectID, error) {

	const location = "build.StepWithFollowing.getUserID"

	// Admin routes use zeroes for "Following" parents
	if builder.IsAdminBuilder() {

		// Must be an owner to use the admin route
		if !builder.IsOwner() {
			return primitive.NilObjectID, derp.ForbiddenError(location, "User must be an owner to complete this action")
		}

		return primitive.NilObjectID, nil
	}

	// Non-admin routes use the authenticated user's ID
	return builder.AuthenticatedID(), nil

}
