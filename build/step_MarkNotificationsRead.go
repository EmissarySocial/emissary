package build

import (
	"io"
	"time"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepMarkNotificationsRead is a Step that marks the authenticated User's notifications as read:
// the one named by the "notificationId" query parameter, or else every one in the "type" tab.
type StepMarkNotificationsRead struct{}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepMarkNotificationsRead) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post marks one of the authenticated User's notifications as read, or a whole tab of them
func (step StepMarkNotificationsRead) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepMarkNotificationsRead.Post"

	// RULE: User MUST be authenticated to use this step
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Anonymous user is not authorized to perform this action"))
	}

	notificationService := builder.factory().Notification()
	readDate := time.Now().Unix()

	// A request that names one Notification marks that one, and nothing else
	if token := builder.QueryParam("notificationId"); token != "" {

		notificationID, err := primitive.ObjectIDFromHex(token)

		if err != nil {
			return Halt().WithError(derp.BadRequest(location, "NotificationID must be a valid hex string"))
		}

		if err := notificationService.MarkRead(builder.session(), builder.AuthenticatedID(), notificationID, readDate); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Marking notification read"))
		}

		return Continue()
	}

	// Otherwise, mark every Notification in the tab that the "type" query parameter names
	types := notificationTabTypes(builder.QueryParam("type"))

	if err := notificationService.MarkAllRead(builder.session(), builder.AuthenticatedID(), readDate, types...); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Marking notifications read"))
	}

	return Continue()
}
