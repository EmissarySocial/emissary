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

func (step StepWithAnnotation) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithAnnotation) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithAnnotation) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithAnnotation.execute"

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.InternalError(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	annotation, err := step.getAnnotation(builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to get Annotation record"))
	}

	// Create a new builder tied to the Annotation record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &annotation, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the build pipeline on the Annotation record
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

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

		if err := annotationService.LoadByURL(builder.session(), userID, url, &annotation); !derp.IsNilOrNotFound(err) {
			return model.NewAnnotation(), derp.Wrap(err, location, "Unable to load Annotation by URL", url)
		}

		annotation.URL = url
		return annotation, nil
	}

	// Otherwise, use the `annotationId` query parameter to load the Annotation record
	token := builder.QueryParam("annotationId")

	// Finally, try to load the Annotation record from the database.
	if err := annotationService.LoadByToken(builder.session(), userID, token, &annotation); err != nil {
		return annotation, derp.Wrap(err, location, "Unable to load Annotation", token)
	}

	// Success.
	return annotation, nil
}
