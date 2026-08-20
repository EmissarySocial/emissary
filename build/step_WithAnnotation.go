package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithAnnotation is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithAnnotation struct {
	SubSteps []step.Step
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepWithAnnotation) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithAnnotation) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

// execute performs this step's work for either a GET or a POST
func (step StepWithAnnotation) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithAnnotation.execute"

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.Internal(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	annotation, err := step.getAnnotation(builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Getting Annotation record"))
	}

	// Create a new builder tied to the Annotation record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &annotation, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Creating sub-builder"))
	}

	// Execute the build pipeline on the Annotation record
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Executing steps for child")

	return UseResult(result)
}

// getAnnotation loads the Annotation named in the query string, or returns a new one
func (step StepWithAnnotation) getAnnotation(builder Builder) (model.Annotation, error) {

	const location = "build.StepWithAnnotation.getAnnotation"

	userID := builder.AuthenticatedID()

	// Collect required services and values
	factory := builder.factory()
	annotationService := factory.Annotation()
	annotation := model.NewAnnotation()
	annotation.UserID = userID

	// If a `url` query parameter is provided, then use it to load the Annotation record
	if url := builder.QueryParam("url"); url != "" {

		if err := annotationService.LoadByURL(builder.session(), userID, url, &annotation); err != nil {
			if !derp.IsNotFound(err) {
				return model.NewAnnotation(), derp.Wrap(err, location, "Loading Annotation by URL", url)
			}
		}

		annotation.URL = url
		return annotation, nil
	}

	// Otherwise, use the `annotationId` query parameter to load the Annotation record
	token := builder.QueryParam("annotationId")

	// Finally, try to load the Annotation record from the database.
	if err := annotationService.LoadByToken(builder.session(), userID, token, &annotation); err != nil {
		return annotation, derp.Wrap(err, location, "Loading Annotation", token)
	}

	// Success.
	return annotation, nil
}
