package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithKeyPackage is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithKeyPackage struct {
	SubSteps []step.Step
}

func (step StepWithKeyPackage) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithKeyPackage) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithKeyPackage) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithKeyPackage.execute"

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

	// Parse the KeyPackageID
	keyPackageID, err := primitive.ObjectIDFromHex(builder.QueryParam("keyPackageId"))

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Invalid KeyPackageID", builder.QueryParam("keyPackageId")))
	}

	// Load the KeyPackage from the database
	factory := builder.factory()
	keyPackageService := factory.KeyPackage()
	keyPackage := model.NewKeyPackage()
	if err := keyPackageService.LoadByID(builder.session(), builder.AuthenticatedID(), keyPackageID, &keyPackage); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to load KeyPackage", keyPackageID))
	}

	// Create a new builder tied to the KeyPackage record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &keyPackage, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}
