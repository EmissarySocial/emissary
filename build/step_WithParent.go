package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
)

// StepWithParent is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithParent struct {
	SubSteps []step.Step
}

func (step StepWithParent) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post executes the subSteps on the parent Stream
func (step StepWithParent) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepWithParent.Post"

	factory := builder.factory()
	streamBuilder := builder.(Stream)

	// If the parent is a USER object, then we have a slightly different workflow
	if streamBuilder._stream.ParentID == streamBuilder.AttributedTo().UserID {
		return step.postUser(streamBuilder, buffer)
	}

	// Otherwise, load the parent Stream...
	var parent model.Stream

	// Try to load the parent Stream
	if err := factory.Stream().LoadByID(builder.session(), streamBuilder._stream.ParentID, &parent); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error listing parent"))
	}

	// Make a builder with the new parent stream
	// TODO: LOW: Is "view" really the best action to use here??
	parentStream, err := NewStreamWithoutTemplate(streamBuilder.factory(), streamBuilder.session(), streamBuilder.request(), streamBuilder.response(), &parent, "")

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error creating builder for parent"))
	}

	// Execute the POST build pipeline on the parent
	result := Pipeline(step.SubSteps).Post(factory, &parentStream, buffer)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for parent")
	return UseResult(result)
}

func (step StepWithParent) postUser(streamBuilder Stream, buffer io.Writer) PipelineBehavior {

	const location = "build.StepWithParent.postUser"

	var user model.User

	log.Trace().Str("parentID", streamBuilder._stream.ParentID.Hex()).Msg("step.WithParent.postUser")

	factory := streamBuilder.factory()

	// Try to load the parent Stream
	if err := factory.User().LoadByID(streamBuilder.session(), streamBuilder._stream.ParentID, &user); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error listing parent"))
	}

	// Make a builder with the new parent stream
	// TODO: LOW: Is "view" really the best action to use here??
	outbox, err := NewOutbox(streamBuilder.factory(), streamBuilder.session(), streamBuilder.request(), streamBuilder.response(), &user, "view")

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error creating builder for parent"))
	}

	// Execute the POST build pipeline on the parent
	result := Pipeline(step.SubSteps).Post(factory, &outbox, buffer)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for parent")

	return UseResult(result)
}
