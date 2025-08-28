package build

import (
	"io"

	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSort is a Step that can update multiple records at once
type StepSort struct {
	Model   string
	Keys    string
	Values  string
	Message string
}

func (step StepSort) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSort) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSort.Post"

	var modelService service.ModelService
	var err error

	// Collect form POST information
	transaction, err := formdata.Parse(builder.request())
	if err != nil {
		return Halt().WithError(derp.BadRequestError(location, "Error binding body"))
	}

	// Locate the model service to use
	if step.Model == "" {
		modelService = builder.service()
	} else {
		modelService, err = builder.factory().Model(step.Model)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error loading model service", step.Model))
		}
	}

	for index, keyID := range transaction["keys"] {

		// Adding one so that our index does not include 0 (rank not set)
		newRank := index + 1

		// Collect inputs to make a selection criteria
		objectID, err := primitive.ObjectIDFromHex(keyID)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Invalid objectId", keyID))
		}

		criteria := exp.Equal(step.Keys, objectID)

		// Try to load the object from the database
		object, err := modelService.ObjectLoad(builder.session(), criteria)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error loading object with criteria: ", criteria))
		}

		// Use the object schema to set the new sort rank
		if err := modelService.Schema().Set(object, step.Values, newRank); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error setting new rank", objectID, step.Values, newRank))
		}

		// Try to save back to the database
		if err := modelService.ObjectSave(builder.session(), object, step.Message); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error saving record tot he database", object))
		}
	}

	// Done. Do not swap on the client side.  We don't want to reload the entire page.
	return Continue().WithHeader("HX-Reswap", "none")
}
