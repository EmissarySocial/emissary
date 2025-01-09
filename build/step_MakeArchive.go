package build

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// StepMakeArchive is a Step that can delete a Stream from the Domain
type StepMakeArchive struct {
	Token       string
	Depth       int
	JSON        bool
	Attachments bool
	Metadata    [][]map[string]any
}

// Get displays a customizable confirmation form for the delete
func (step StepMakeArchive) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return Continue()
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepMakeArchive) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepMakeArchive.Get"

	// Guarantee that we have a Stream builder
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		err := derp.NewBadRequestError(location, "The `export` step can only be called on a `Stream` builder")
		return Halt().WithError(err)
	}

	// Add a Task to the Queue
	q := streamBuilder.factory().Queue()
	task := queue.NewTask("MakeStreamArchive", mapof.Any{
		"host":        streamBuilder.Hostname(),
		"streamId":    streamBuilder.StreamID(),
		"token":       step.Token,
		"depth":       step.Depth,
		"json":        step.JSON,
		"attachments": step.Attachments,
		"metadata":    step.Metadata,
	})

	if err := q.Publish(task); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error publishing task", task))
	}

	// Success
	return Continue()
}
