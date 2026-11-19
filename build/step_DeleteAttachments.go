package build

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/convert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepDeleteAttachments represents an action that can delete one or more attachments.  It can only be used on a StreamBuilder
type StepDeleteAttachments struct {
	All      bool
	Field    string
	Category string
	Minimum  int
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepDeleteAttachments) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepDeleteAttachments) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepDeleteAttachments.Post"

	factory := builder.factory()

	attachmentService := factory.Attachment()

	objectType := builder.service().ObjectType()
	objectID := builder.objectID()

	var criteria exp.Expression = exp.All()

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
			return Halt().WithError(derp.Wrap(err, location, "Clearing field value"))
		}

	} else {

		// Filter on all attachments within the designated Category
		if step.Category != "" {
			criteria = criteria.AndEqual("category", step.Category)
		}

		// RULE: An attachmentId that is present must be an ID. Dropping a bad one from the
		// criteria would widen the delete from one file to the whole category.
		if attachmentIDString := builder.QueryParam("attachmentId"); attachmentIDString != "" {

			attachmentID, err := primitive.ObjectIDFromHex(attachmentIDString)

			if err != nil {
				return Halt().WithError(derp.BadRequest(location, "Invalid attachmentId", attachmentIDString))
			}

			criteria = criteria.AndEqual("_id", attachmentID)
		}

		// Require that there is at least one criteria OR that "ALL" has been specified.
		// If the criteria is empty, then NOOP.
		if criteria.IsEmpty() && !step.All {
			return Continue()
		}

		// RULE: The category must keep at least the minimum, so a delete that would leave fewer is refused whole
		if step.Minimum > 0 {

			remaining, err := remainingAfterDelete(builder, objectType, objectID, step.Category, criteria)

			if err != nil {
				return Halt().WithError(derp.Wrap(err, location, "Counting attachments"))
			}

			if remaining < step.Minimum {
				return Halt().WithError(derp.BadRequest(location, minimumMessage(step.Minimum), "minimum", step.Minimum, "remaining", remaining))
			}
		}
	}

	// Delete the attachments that match the object and criteria
	if err := attachmentService.DeleteByCriteria(builder.session(), objectType, objectID, criteria, "Deleted by Workflow Step"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Deleting all attachments"))
	}

	// Notify the client
	return Continue().WithEvent("attachments-updated", "true")
}

// remainingAfterDelete returns how many attachments the category would still hold once the
// ones matching the criteria are deleted
func remainingAfterDelete(builder Builder, objectType string, objectID primitive.ObjectID, category string, criteria exp.Expression) (int, error) {

	const location = "build.remainingAfterDelete"

	attachmentService := builder.factory().Attachment()

	holding, err := attachmentService.CountByCategory(builder.session(), objectType, objectID, category)

	if err != nil {
		return 0, derp.Wrap(err, location, "Counting the category", objectType, objectID, category)
	}

	deleting, err := attachmentService.CountByCriteria(builder.session(), objectType, objectID, criteria)

	if err != nil {
		return 0, derp.Wrap(err, location, "Counting the attachments to delete", objectType, objectID)
	}

	return holding - deleting, nil
}
