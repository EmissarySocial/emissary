package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepWithCollectionItem is a Step that runs its sub-steps on one CollectionItem that the authenticated
// User owns.  The "collectionItemId" query parameter names the item, which must be of the step's Type.
type StepWithCollectionItem struct {
	Type     string
	SubSteps []step.Step
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepWithCollectionItem) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post runs the sub-steps on the CollectionItem during a POST request. Implements the Step interface.
func (step StepWithCollectionItem) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

// execute performs this step's work for either a GET or a POST
func (step StepWithCollectionItem) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithCollectionItem.execute"

	// RULE: User MUST be authenticated to use this step
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Anonymous user is not authorized to perform this action"))
	}

	// Find the Template for this builder, which every builder that uses CommonWithTemplate has
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.Internal(location, "This step cannot be used in this Renderer."))
	}

	// Load the CollectionItem that the request names
	collectionItem, err := step.getCollectionItem(builder, actionMethod)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Getting CollectionItem"))
	}

	// Create a new builder tied to the CollectionItem
	factory := builder.factory()
	subBuilder, err := NewModel(factory, builder.session(), builder.request(), builder.response(), template, &collectionItem, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Creating sub-builder"))
	}

	// Execute the sub-steps on the CollectionItem
	result := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	result.Error = derp.WrapIF(result.Error, location, "Executing steps for CollectionItem")

	return UseResult(result)
}

// getCollectionItem loads the CollectionItem named in the query string, which the authenticated User must own.
// On a POST, an item that is already gone comes back empty, so that deleting it again still succeeds.
func (step StepWithCollectionItem) getCollectionItem(builder Builder, actionMethod ActionMethod) (model.CollectionItem, error) {

	const location = "build.StepWithCollectionItem.getCollectionItem"

	result := model.NewCollectionItem()

	// Parse the CollectionItemID from the query string
	token := builder.QueryParam("collectionItemId")
	collectionItemID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return result, derp.BadRequest(location, "CollectionItemID must be a valid hex string", token)
	}

	// RULE: The CollectionItem must belong to the authenticated User
	userID := builder.AuthenticatedID()

	if err := builder.factory().CollectionItem().LoadByUserID(builder.session(), userID, collectionItemID, &result); err != nil {

		// Anything other than "not found" is a genuine failure to reach the database
		if !derp.IsNotFound(err) {
			return result, derp.Wrap(err, location, "Loading CollectionItem", collectionItemID)
		}

		// There is nothing to show for an item that does not exist
		if actionMethod == ActionMethodGet {
			return result, derp.Wrap(err, location, "Loading CollectionItem", collectionItemID)
		}

		// A POST gets an empty item of the right type, which names nothing but the User and the ID
		result.CollectionItemID = collectionItemID
		result.UserID = userID
		result.CollectionType = step.Type
		return result, nil
	}

	// RULE: This step works on one collection type only.  Any other item is reported as missing
	if result.CollectionType != step.Type {
		return result, derp.NotFound(location, "CollectionItem is not of the required type", collectionItemID, step.Type)
	}

	// Collect them all!
	return result, nil
}
