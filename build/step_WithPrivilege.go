package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithPrivilege is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithPrivilege struct {
	SubSteps []step.Step
}

func (step StepWithPrivilege) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithPrivilege) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithPrivilege) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithPrivilege.doStep"

	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.UnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.InternalError(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	privilegeService := factory.Privilege()
	privilegeToken := builder.QueryParam("privilegeId")
	privilege := model.NewPrivilege()
	privilege.UserID = builder.AuthenticatedID()

	if circleID, identityID, exists := step.getCircleAndIdentity(builder); exists {

		if err := privilegeService.LoadByIdentityAndCircle(builder.session(), builder.AuthenticatedID(), identityID, circleID, &privilege); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Unable to load Privilege by Identity and Circle", "identityID: "+identityID.Hex(), "circleID: "+circleID.Hex()))
		}
	}

	if (privilegeToken != "") && (privilegeToken != "new") {

		privilegeID, err := primitive.ObjectIDFromHex(privilegeToken)
		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Invalid Privilege ID", privilegeToken))
		}

		if err := privilegeService.LoadByID(builder.session(), builder.AuthenticatedID(), privilegeID, &privilege); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Privilege", privilegeID))
			}
			// Fall through for POSTS..  we're just creating a new privilege.
		}
	}

	// Create a new builder tied to the Privilege record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &privilege, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.Wrap(result.Error, location, "Error executing steps for child")

	return UseResult(result)
}

// Special case to find/load privileges using the CircleID and IdentityID
func (step StepWithPrivilege) getCircleAndIdentity(builder Builder) (primitive.ObjectID, primitive.ObjectID, bool) {

	// Does the URL request have a circleId?
	circleToken := builder.QueryParam("circleId")

	if circleToken == "" {
		return primitive.NilObjectID, primitive.NilObjectID, false
	}

	// Is the circleId a valid ObjectID?
	circleID, err := primitive.ObjectIDFromHex(circleToken)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, false
	}

	// Does the URL request have an identityId?
	identityToken := builder.QueryParam("identityId")

	if identityToken == "" {
		return circleID, primitive.NilObjectID, false
	}

	// Is the identityId a valid ObjectID?
	identityID, err := primitive.ObjectIDFromHex(identityToken)

	if err != nil {
		return circleID, primitive.NilObjectID, false
	}

	// Then YES.  We have a CircleID and an IdentityID
	return circleID, identityID, true
}
