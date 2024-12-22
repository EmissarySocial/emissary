package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSortAttachments is an action-step that can update multiple records at once
type StepSortAttachments struct {
	Keys    string
	Values  string
	Message string
}

func (step StepSortAttachments) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepSortAttachments) Post(builder Builder, _ io.Writer) PipelineBehavior {

	var transaction struct {
		Keys []string `form:"keys"`
	}

	// Collect form POST information
	if err := bind(builder.request(), &transaction); err != nil {
		return Halt().WithError(derp.NewBadRequestError("build.StepSortAttachments.Post", "Error binding body"))
	}

	factory := builder.factory()
	attachmentService := factory.Attachment()

	for index, id := range transaction.Keys {

		var attachment model.Attachment
		newRank := index + 1 // Adding one so that ranks don't include 0 (rank unset)

		// Collect inputs to make a selection criteria
		attachmentID, err := primitive.ObjectIDFromHex(id)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, "build.StepSortAttachments.Post", "Invalid attachmentId", id))
		}

		criteria := exp.Equal("streamId", builder.objectID()).
			AndEqual(step.Keys, attachmentID).
			AndEqual("deleteDate", 0)

		// Try to load the attachment from the database
		if err := attachmentService.Load(criteria, &attachment); err != nil {
			return Halt().WithError(derp.Wrap(err, "build.StepSortAttachments.Post", "Error loading attachment with criteria: ", criteria))
		}

		// If the rank for this attachment has not changed, then don't waste time saving it again.
		if attachment.Rank == newRank {
			continue
		}

		attachment.Rank = newRank

		// Try to save back to the database
		if err := attachmentService.Save(&attachment, step.Message); err != nil {
			return Halt().WithError(derp.Wrap(err, "build.StepSortAttachments.Post", "Error saving record tot he database", attachment))
		}
	}

	// Done.  Nothing more to do here.
	return nil
}
