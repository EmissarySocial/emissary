package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithOAuthToken is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithOAuthToken struct {
	SubSteps []step.Step
}

func (step StepWithOAuthToken) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithOAuthToken) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithOAuthToken) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithOAuthToken.execute"

	// RULE: User MUST be authenticated to use this step
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Anonymous user is not authorized to perform this action"))
	}

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.Internal(location, "This step cannot be used in this Renderer."))
	}

	// Parse the OAuthUserTokenID
	oauthUserTokenID, err := primitive.ObjectIDFromHex(builder.QueryParam("oauthUserTokenId"))

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Invalid OAuthUserTokenID", builder.QueryParam("oauthUserTokenId")))
	}

	// Load the OAuthUserToken from the database
	factory := builder.factory()
	oauthUserTokenService := factory.OAuthUserToken()
	oauthUserToken := model.NewOAuthUserToken()
	if err := oauthUserTokenService.LoadByID(builder.session(), builder.AuthenticatedID(), oauthUserTokenID, &oauthUserToken); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to load OAuthUserToken", oauthUserTokenID))
	}

	// Create a new builder tied to the OAuthToken record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &oauthUserToken, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}
