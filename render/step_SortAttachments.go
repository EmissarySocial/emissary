package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSortAttachments represents an action-step that can update multiple records at once
type StepSortAttachments struct {
	Keys    string
	Values  string
	Message string
}

func (step StepSortAttachments) Get(renderer Renderer, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSortAttachments) Post(renderer Renderer, _ io.Writer) PipelineBehavior {

	var formPost struct {
		Keys []string `form:"keys"`
	}

	// Collect form POST information
	if err := renderer.context().Bind(&formPost); err != nil {
		return Halt().WithError(derp.NewBadRequestError("render.StepSortAttachments.Post", "Error binding body"))
	}

	factory := renderer.factory()
	attachmentService := factory.Attachment()

	for index, id := range formPost.Keys {

		var attachment model.Attachment
		newRank := index + 1 // Adding one so that ranks don't include 0 (rank unset)

		// Collect inputs to make a selection criteria
		attachmentID, err := primitive.ObjectIDFromHex(id)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, "render.StepSortAttachments.Post", "Invalid attachmentId", id))
		}

		criteria := exp.Equal("streamId", renderer.objectID()).
			AndEqual(step.Keys, attachmentID).
			AndEqual("deleteDate", 0)

		// Try to load the attachment from the database
		if err := attachmentService.Load(criteria, &attachment); err != nil {
			return Halt().WithError(derp.Wrap(err, "render.StepSortAttachments.Post", "Error loading attachment with criteria: ", criteria))
		}

		// If the rank for this attachment has not changed, then don't waste time saving it again.
		if attachment.Rank == newRank {
			continue
		}

		attachment.Rank = newRank

		// Try to save back to the database
		if err := attachmentService.Save(&attachment, step.Message); err != nil {
			return Halt().WithError(derp.Wrap(err, "render.StepSortAttachments.Post", "Error saving record tot he database", attachment))
		}
	}

	// Done.  Nothing more to do here.
	return nil
}
