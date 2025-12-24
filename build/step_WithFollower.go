package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithFollower is a Step that can update the data.DataMap custom data stored in a Stream
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
		return Halt().WithError(derp.InternalError(location, "This step cannot be used in this Renderer."))
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

	// Collect required services and values
	factory := builder.factory()
	follower := model.NewFollower()
	follower.ParentID = userID // TODO: Make this generic enough to work with Content Actors...

	// Only authenticated users can create new Follower records
	if token := builder.QueryParam("followerId"); isNewOrEmpty(token) {

		if !builder.IsAuthenticated() {
			return Halt().WithError(derp.ForbiddenError(location, "Anonymous user is not authorized to perform this action"))
		}

	} else {

		// Existing Follower records can be looked up if authenticated, or with a secret.
		followerService := factory.Follower()
		secret := builder.QueryParam("secret")
		if err := step.load(builder.session(), followerService, token, userID, secret, &follower); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Unable to load Follower via ID", token))
		}
	}

	// Create a new builder tied to the Follower record
	subBuilder, err := NewFollower(factory, builder.session(), builder.request(), builder.response(), template, &follower, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the build pipeline on the Follower record
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

func (step StepWithFollower) load(session data.Session, followerService *service.Follower, token string, authenticatedID primitive.ObjectID, secret string, follower *model.Follower) error {

	const location = "build.StepWithFollower.load"

	// If a secret is present, then use it to load the Follower record
	if secret != "" {

		followerID, err := primitive.ObjectIDFromHex(token)

		if err != nil {
			return derp.Wrap(err, location, "Invalid follower ID", token)
		}

		return followerService.LoadBySecret(session, followerID, secret, follower)
	}

	// If the user is Authenticated, then use their Authentication to load the Follower record.
	if !authenticatedID.IsZero() {
		return followerService.LoadByToken(session, authenticatedID, token, follower)
	}

	// Nope.  Not happening.
	return derp.ForbiddenError(location, "Anonymous user is not authorized to perform this action")

}
