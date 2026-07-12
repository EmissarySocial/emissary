package build

import (
	"io"
	"time"

	"github.com/benpate/derp"
)

// StepMarkNotificationsRead is a Step that marks all of the authenticated User's notifications as read.
type StepMarkNotificationsRead struct{}

func (step StepMarkNotificationsRead) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post marks all of the authenticated User's notifications as read.
func (step StepMarkNotificationsRead) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepMarkNotificationsRead.Post"

	// RULE: User MUST be authenticated to use this step
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Anonymous user is not authorized to perform this action"))
	}

	notificationService := builder.factory().Notification()

	if err := notificationService.MarkAllRead(builder.session(), builder.AuthenticatedID(), time.Now().Unix()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to mark notifications read"))
	}

	return Continue()
}
