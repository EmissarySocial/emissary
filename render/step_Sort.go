package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/path"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSort represents an action-step that can update multiple records at once
type StepSort struct {
	modelService ModelService
	keys         string
	values       string
	message      string
}

func NewStepSort(modelService ModelService, stepInfo datatype.Map) StepSort {

	return StepSort{
		modelService: modelService,
		keys:         stepInfo.GetString("keys"),
		values:       stepInfo.GetString("values"),
		message:      stepInfo.GetString("message"),
	}
}

// Get does not display anything.
func (step StepSort) Get(buffer io.Writer, renderer Renderer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSort) Post(buffer io.Writer, renderer Renderer) error {

	var formPost struct {
		Keys []string `form:"keys"`
	}

	// Collect form POST information
	if err := renderer.context().Bind(&formPost); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepSort.Post", "Error binding body")
	}

	for rank, key := range formPost.Keys {

		// Collect inputs to make a selection criteria
		objectId, err := primitive.ObjectIDFromHex(key)

		if err != nil {
			return derp.Wrap(err, "ghost.render.StepSort.Post", "Invalid objectId", key)
		}

		criteria := exp.Equal(step.keys, objectId)

		// Try to load the object from the database
		object, err := step.modelService.ObjectLoad(criteria)

		if err != nil {
			return derp.Wrap(err, "ghost.render.StepSort.Post", "Error loading object with criteria: ", criteria)
		}

		// Update the object
		path.Set(object, key, rank)

		// Try to save back to the database
		if err := step.modelService.ObjectSave(object, step.message); err != nil {
			return derp.Wrap(err, "ghost.render.StepSort.Post", "Error saving record tot he database", object)
		}
	}

	/*
		// Put approved form data into the stream
		for _, p := range step.paths {
			if err := renderer.SetPath(path.New(p), inputs[p]); err != nil {
				return derp.New(derp.CodeBadRequestError, "ghost.render.StepSort.Post", "Error seting value from user input", p)
			}
		}

		// Put values from schema.json into the stream
		for key, value := range step.values {
			if err := renderer.SetPath(path.New(key), value); err != nil {
				return derp.Wrap(err, "ghose.render.StepSort.Post", "Error setting value from schema.json", key, value)
			}
		}
	*/
	return nil
}
