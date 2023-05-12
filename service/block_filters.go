package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

/******************************************
 * Filters
 ******************************************/

func (service *Block) FilterFollower(follower *model.Follower) error {

	// RULE: Blocks ONLY work on ActivityPub followers
	if follower.Method != model.FollowMethodActivityPub {
		return nil
	}

	// Get a list of all blocks for this User
	activeBlocks, err := service.QueryActiveByUser(follower.ParentID)

	if err != nil {
		return derp.Wrap(err, "service.Block.FilterFollower", "Error loading blocks for user", follower.ParentID)
	}

	// Try each block. If "BLOCK", then do not allow the follower
	for _, block := range activeBlocks {
		if block.FilterByActor(follower.Actor.ProfileURL) {
			return derp.NewValidationError("Actor blocked")
		}
	}

	// No block means that this follower is allowed
	return nil
}

func (service *Block) FilterMention(mention *model.Mention) error {

	// Get a list of all blocks for this User
	activeBlocks, err := service.QueryActiveByUser(mention.ObjectID)

	if err != nil {
		return derp.Wrap(err, "service.Block.FilterFollower", "Error loading blocks for user", mention.ObjectID)
	}

	// Try each block.  If "BLOCK" or "MUTE", then do not allow the mention
	for _, block := range activeBlocks {
		if block.FilterByActors(mention.Origin.URL, mention.Author.ProfileURL) {
			return derp.NewValidationError("Actor blocked")
		}
	}

	// No block means that this follower is allowed
	return nil
}

func (service *Block) FilterResponse(response *model.Response) error {

	// Get a list of all blocks for this User
	userID := response.Object.AttributedTo.First().UserID
	activeBlocks, err := service.QueryActiveByUser(userID)

	if err != nil {
		return derp.Wrap(err, "service.Block.FilterFollower", "Error loading blocks for user", userID)
	}

	// Try each block.  If "BLOCK" or "MUTE", then do not allow the mention
	for _, block := range activeBlocks {
		if block.FilterByActors(response.Origin.URL, response.Actor.ProfileURL) {
			return derp.NewValidationError("Actor blocked")
		}
	}

	// No block means that this follower is allowed
	return nil
}

func (service *Block) FilterMessage(message *model.Message) error {

	// Get a list of all blocks for this User
	activeBlocks, err := service.QueryActiveByUser(message.UserID)

	if err != nil {
		return derp.Wrap(err, "service.Block.filterMessage", "Error loading blocks for user", message.UserID)
	}

	// Try to execute each block
	for _, block := range activeBlocks {
		if block.FilterByActorAndContent(message.Origin.URL, message.Document.Label, message.Document.Summary, message.ContentHTML) {
			return derp.NewValidationError("Actor blocked")
		}
	}

	return nil
}
