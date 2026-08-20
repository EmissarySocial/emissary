package build

import (
	"io"
	"time"

	"github.com/benpate/derp"
)

// StepMarkNotificationsRead is a Step that marks the authenticated User's notifications as read.
// The optional "type" query parameter scopes the update to one notifications-page tab (using
// the same type expansion as the tab filter); without it, ALL notifications are marked read.
type StepMarkNotificationsRead struct{}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepMarkNotificationsRead) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post marks the authenticated User's notifications (per the "type" query parameter) as read.
func (step StepMarkNotificationsRead) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepMarkNotificationsRead.Post"

	// RULE: User MUST be authenticated to use this step
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Anonymous user is not authorized to perform this action"))
	}

	notificationService := builder.factory().Notification()
	types := notificationTabTypes(builder.QueryParam("type"))

	if err := notificationService.MarkAllRead(builder.session(), builder.AuthenticatedID(), time.Now().Unix(), types...); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Marking notifications read"))
	}

	return Continue()
}
