package build

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/convert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepDeleteAttachments represents an action that can delete ine or more attachments.  It can only be used on a StreamBuilder
type StepDeleteAttachments struct {
	All      bool
	Field    string
	Category string
}

func (step StepDeleteAttachments) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

func (step StepDeleteAttachments) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "builder.StepDeleteAttachments.Post"

	factory := builder.factory()

	attachmentService := factory.Attachment()

	objectType := builder.service().ObjectType()
	objectID := builder.objectID()

	criteria := exp.All()

	// If "field" is specified, then all other arguments are ignored.
	if step.Field != "" {

		// Look up the value of the designated field, convert to an ObjectID, and add to the criteria
		s := builder.schema()
		if value, err := s.Get(builder.object(), step.Field); err == nil {
			if valueString := convert.String(value); valueString != "" {
				if valueID, err := primitive.ObjectIDFromHex(valueString); err == nil {
					criteria = criteria.AndEqual("_id", valueID)
				}
			}
		}

		// If field value could not be resolved, then NOOP
		if criteria.IsEmpty() {
			return Continue()
		}

		// Clear the value from the attachment field
		if err := s.Set(builder.object(), step.Field, ""); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error clearing field value"))
		}

	} else {

		// Filter on all attachments within the designated Category
		if step.Category != "" {
			criteria = criteria.AndEqual("category", step.Category)
		}

		// Filter on the AttachmentID in the query parameter
		if attachmentIDString := builder.QueryParam("attachmentId"); attachmentIDString != "" {

			if attachmentID, err := primitive.ObjectIDFromHex(attachmentIDString); err == nil {
				criteria = criteria.AndEqual("_id", attachmentID)
			}
		}

		// Require that there is at least one criteria OR that "ALL" has been specified.
		// If the criteria is empty, then NOOP.
		if criteria.IsEmpty() && !step.All {
			return Continue()
		}
	}

	// Delete the attachments that match the object and criteria
	if err := attachmentService.DeleteByCriteria(objectType, objectID, criteria, "Deleted by Workflow Step"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error deleting all attachments"))
	}

	// Notify the client
	builder.response().Header().Set("HX-Trigger", `attachments-updated`)

	return nil
}
