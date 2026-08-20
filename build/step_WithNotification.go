package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithNotification is a Step that executes a new pipeline on a Notification, identified by the
// query parameter "notificationId" and scoped to the authenticated User.
type StepWithNotification struct {
	SubSteps []step.Step
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepWithNotification) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the notification with data from the request body.
func (step StepWithNotification) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

// execute performs this step's work for either a GET or a POST
func (step StepWithNotification) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithNotification.execute"

	// RULE: User MUST be authenticated to use this step
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Anonymous user is not authorized to perform this action"))
	}

	// Try to find the Template for this builder.
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.Internal(location, "This step cannot be used in this Renderer."))
	}

	// Parse the NotificationID from the query string
	notificationID, err := primitive.ObjectIDFromHex(builder.QueryParam("notificationId"))

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "NotificationID must be a valid hex string"))
	}

	// Collect required services and values
	factory := builder.factory()
	notificationService := factory.Notification()
	notification := model.NewNotification()
	userID := builder.AuthenticatedID()

	// Load the notification, scoped to the authenticated User (LoadByID enforces ownership).
	if err := notificationService.LoadByID(builder.session(), userID, notificationID, &notification); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Loading Notification", notificationID))
	}

	// Create a new builder tied to the Notification record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &notification, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Creating sub-builder"))
	}

	// Execute the build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Executing steps for child")

	return UseResult(result)
}
