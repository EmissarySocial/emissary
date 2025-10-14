package build

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// StepScheduleDelete is a Step that forwards the user to a new page.
type StepScheduleDelete struct {
	Days    *template.Template
	Hours   *template.Template
	Minutes *template.Template
	Seconds *template.Template
}

func (step StepScheduleDelete) Get(_ Builder, _ io.Writer) PipelineBehavior {
	return Continue()
}

// Post updates the stream with approved data from the request body.
func (step StepScheduleDelete) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepScheduleDelete.Post"

	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.InternalError(location, "StepScheduleDelete can only be used in a Stream context"))
	}

	// Calculate the total time to wait before deleting the Stream (in seconds)
	days := convert.Int(executeTemplate(step.Days, builder))
	hours := convert.Int(executeTemplate(step.Hours, builder))
	minutes := convert.Int(executeTemplate(step.Minutes, builder))
	seconds := convert.Int(executeTemplate(step.Seconds, builder))

	delaySeconds := seconds + (minutes * 60) + (hours * 3600) + (days * 86400)

	// Connect to the task queue
	q := builder.factory().Queue()
	signature := "DELETE:" + streamBuilder.StreamID()

	// Remove existing scheduled delete for this stream (if present)
	if err := q.Delete(signature); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to delete existing task", signature))
	}

	// If the offset time is zero then we don't need to schedule anything else. Exit
	if delaySeconds == 0 {
		return Continue()
	}

	// Schedule the task to delete this stream after [delaySeconds]
	q.NewTask(
		"DeleteStream",
		mapof.Any{
			"streamId": streamBuilder.StreamID(),
		},
		queue.WithSignature(signature),
		queue.WithDelaySeconds(delaySeconds),
	)

	// Success
	return Continue()
}
