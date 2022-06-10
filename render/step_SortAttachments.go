package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSortAttachments represents an action-step that can update multiple records at once
type StepSortAttachments struct {
	Keys    string
	Values  string
	Message string
}

func (step StepSortAttachments) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepSortAttachments) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepSortAttachments) Post(renderer Renderer) error {

	var formPost struct {
		Keys []string `form:"keys"`
	}

	// Collect form POST information
	if err := renderer.context().Bind(&formPost); err != nil {
		return derp.NewBadRequestError("render.StepSortAttachments.Post", "Error binding body")
	}

	factory := renderer.factory()
	attachmentService := factory.Attachment()

	for rank, id := range formPost.Keys {

		var attachment model.Attachment

		// Collect inputs to make a selection criteria
		attachmentID, err := primitive.ObjectIDFromHex(id)

		if err != nil {
			return derp.Wrap(err, "render.StepSortAttachments.Post", "Invalid attachmentId", id)
		}

		criteria := exp.Equal("streamId", renderer.objectID()).
			AndEqual(step.Keys, attachmentID).
			AndEqual("journal.deleteDate", 0)

		// Try to load the attachment from the database
		if err := attachmentService.Load(criteria, &attachment); err != nil {
			return derp.Wrap(err, "render.StepSortAttachments.Post", "Error loading attachment with criteria: ", criteria)
		}

		// If the rank for this attachment has not changed, then don't waste time saving it again.
		if attachment.Rank == rank {
			continue
		}

		attachment.Rank = rank

		// Try to save back to the database
		if err := attachmentService.Save(&attachment, step.Message); err != nil {
			return derp.Wrap(err, "render.StepSortAttachments.Post", "Error saving record tot he database", attachment)
		}
	}

	// Done.  Nothing more to do here.
	return nil
}
