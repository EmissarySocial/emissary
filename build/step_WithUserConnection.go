package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithUserConnection is a Step that scopes its sub-steps to one of the current User's external connections
type StepWithUserConnection struct {
	SubSteps []step.Step
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepWithUserConnection) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepWithUserConnection) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

// execute performs this step's work for either a GET or a POST
func (step StepWithUserConnection) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithUserConnection.execute"

	// RULE: a connection holds a credential that acts on one User's behalf, so it is only
	// ever reachable by that signed-in User
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Anonymous user is not authorized to perform this action"))
	}

	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.Internal(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	userConnection := model.NewUserConnection()
	userConnection.UserID = builder.AuthenticatedID()
	userConnection.Type = builder.QueryParam("type")

	// Load the existing connection, addressed either by its own ID or by the service it reaches
	if err := step.load(builder, &userConnection); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Loading UserConnection"))
	}

	// Create a new builder tied to the UserConnection record
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &userConnection, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Creating sub-builder"))
	}

	// Execute the build pipeline on the child
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Executing steps for child")

	return UseResult(result)
}

// load reads the UserConnection named by the request, by ID or by the service it reaches
func (step StepWithUserConnection) load(builder Builder, userConnection *model.UserConnection) error {

	const location = "build.StepWithUserConnection.load"

	// RULE: every lookup is scoped to the signed-in User, so a guessed ID from another
	// account resolves to nothing rather than to somebody else's credential.
	userConnectionService := builder.factory().UserConnection()
	userID := builder.AuthenticatedID()

	// Addressed by its own ID: the record must exist
	if token := builder.QueryParam("userConnectionId"); notNewOrEmpty(token) {
		return userConnectionService.LoadByUserAndToken(builder.session(), userID, token, userConnection)
	}

	// Addressed by service: a miss means the User has not connected this one yet, which is
	// the state the setup form exists to change.
	if connectionType := builder.QueryParam("type"); connectionType != "" {

		err := userConnectionService.LoadByUserAndType(builder.session(), userID, connectionType, userConnection)

		if derp.IsNotFound(err) {
			return nil
		}

		return err
	}

	return derp.BadRequest(location, "Please name a connection to edit")
}
