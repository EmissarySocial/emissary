package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSetSharing is a Step that forces a Stream's sharing settings to one of the magic Groups
type StepSetSharing struct {
	Role  string
	Group string
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepSetSharing) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post forces the Stream's sharing settings for the configured Role to the configured magic Group
func (step StepSetSharing) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetSharing.Post"

	// RULE: This step can only be applied to a Stream builder
	streamBuilder, isStreamBuilder := builder.(Stream)

	if !isStreamBuilder {
		return Halt().WithError(derp.BadRequest(location, "Builder is not a StreamBuilder"))
	}

	// Overwrite any previous sharing settings for this Role with the configured magic Group
	streamBuilder._stream.Groups[step.Role] = id.Slice{step.groupID()}

	// Sharing is caring
	return nil
}

// groupID maps the step's Group name onto the magic GroupID it identifies
func (step StepSetSharing) groupID() primitive.ObjectID {

	switch step.Group {

	case model.MagicRoleAnonymous:
		return model.MagicGroupIDAnonymous

	case model.MagicRoleAuthenticated:
		return model.MagicGroupIDAuthenticated
	}

	// "owner" is the only other Group name the parser accepts
	return model.MagicGroupIDOwners
}
