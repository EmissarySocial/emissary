package render

import (
	"io"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/first"
	"github.com/benpate/path"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSort represents an action-step that can update multiple records at once
type StepSort struct {
	keys    string
	values  string
	message string
}

func NewStepSort(stepInfo datatype.Map) StepSort {

	return StepSort{
		keys:    first.String(stepInfo.GetString("keys"), "_id"),
		values:  first.String(stepInfo.GetString("values"), "rank"),
		message: stepInfo.GetString("message"),
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
		return derp.New(derp.CodeBadRequestError, "whisper.render.StepSort.Post", "Error binding body")
	}

	for rank, id := range formPost.Keys {

		// Collect inputs to make a selection criteria
		objectID, err := primitive.ObjectIDFromHex(id)

		if err != nil {
			return derp.Wrap(err, "whisper.render.StepSort.Post", "Invalid objectId", id)
		}

		criteria := exp.Equal(step.keys, objectID)

		// Try to load the object from the database
		object, err := renderer.service().ObjectLoad(criteria)

		if err != nil {
			return derp.Wrap(err, "whisper.render.StepSort.Post", "Error loading object with criteria: ", criteria)
		}

		// If the rank for this object has not changed, then don't waste time saving it again.
		if convert.Int(path.Get(object, step.values)) == rank {
			continue
		}

		// Update the object
		if err := path.Set(object, step.values, rank); err != nil {
			return derp.Wrap(err, "whisper.render.StepSort.Post", "Error updating field: ", objectID, rank)
		}

		// Try to save back to the database
		if err := renderer.service().ObjectSave(object, step.message); err != nil {
			return derp.Wrap(err, "whisper.render.StepSort.Post", "Error saving record tot he database", object)
		}

	}

	// Done.  Nothing more to do here.
	return nil
}
