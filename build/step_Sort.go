package build

import (
	"io"

	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/convert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSort is a Step that can update multiple records at once
type StepSort struct {
	Model     string
	Keys      string
	Values    string
	Direction string
	Message   string
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepSort) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post re-ranks the records named in the request body, in the order they were posted.
func (step StepSort) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSort.Post"

	var modelService service.ModelService
	var err error

	// Collect form POST information
	transaction, err := formdata.Parse(builder.request())
	if err != nil {
		return Halt().WithError(derp.BadRequest(location, "Error binding body"))
	}

	// Locate the model service to use
	if step.Model == "" {
		modelService = builder.service()
	} else {
		modelService, err = builder.factory().Model(step.Model)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Loading model service", step.Model))
		}
	}

	// Load every record before saving any, because a descending sort reads their current ranks
	keys := transaction["keys"]
	objects := make([]data.Object, len(keys))
	oldRanks := make([]int, len(keys))

	for index, keyID := range keys {

		objectID, err := primitive.ObjectIDFromHex(keyID)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Invalid objectId", keyID))
		}

		object, err := modelService.ObjectLoad(builder.session(), exp.Equal(step.Keys, objectID))

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Loading object", objectID))
		}

		oldRank, err := modelService.Schema().Get(object, step.Values)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Reading current rank", objectID, step.Values))
		}

		objects[index] = object
		oldRanks[index] = convert.Int(oldRank)
	}

	// Save each record whose rank has changed
	for index, newRank := range sortRanks(oldRanks, step.Direction) {

		if newRank == oldRanks[index] {
			continue
		}

		object := objects[index]

		if err := modelService.Schema().Set(object, step.Values, newRank); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Setting new rank", object.ID(), step.Values, newRank))
		}

		if err := modelService.ObjectSave(builder.session(), object, step.Message); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Saving record to the database", object.ID()))
		}
	}

	// Done. Do not swap on the client side.  We don't want to reload the entire page.
	return Continue().WithHeader("HX-Reswap", "none")
}

// sortRanks returns the new rank for each record, given their current ranks in their new order.
// Ranks never reach 0, which means "not set".
func sortRanks(oldRanks []int, direction string) []int {

	result := make([]int, len(oldRanks))

	// Ascending ranks count up from 1
	if direction != "desc" {
		for index := range result {
			result[index] = index + 1
		}
		return result
	}

	// Descending ranks count down from the highest current rank, so a partial list
	// (the first pages of an infinite scroll) stays above the records not posted
	top := len(oldRanks)

	for _, rank := range oldRanks {
		top = max(top, rank)
	}

	for index := range result {
		result[index] = top - index
	}

	// Some days you're the bug, some days you're the windshield
	return result
}
