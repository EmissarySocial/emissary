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
	blocks, err := service.QueryByUser(follower.ParentID)

	if err != nil {
		return derp.Wrap(err, "service.Block.FilterFollower", "Error loading blocks for user", follower.ParentID)
	}

	// Try each block. If "BLOCK", then do not allow the follower
	for _, block := range blocks {
		if block.FilterByActor(follower.Actor.ProfileURL) {
			if block.Behavior == model.BlockBehaviorBlock {
				return derp.NewValidationError("Actor blocked")
			}
		}
	}

	// No block means that this follower is allowed
	return nil
}

func (service *Block) FilterMention(mention *model.Mention) error {

	// behavior, err := service.filter(mention.UserID, mention.Origin, mention.Document, mention.ContentHTML)

	// Get a list of all blocks for this User
	blocks, err := service.QueryByUser(mention.ObjectID)

	if err != nil {
		return derp.Wrap(err, "service.Block.FilterFollower", "Error loading blocks for user", mention.ObjectID)
	}

	// Try each block.  If "BLOCK" or "MUTE", then do not allow the mention
	for _, block := range blocks {
		if block.FilterByActors(mention.Origin.URL, mention.Author.ProfileURL) {
			if block.Behavior != model.BlockBehaviorAllow {
				return derp.NewValidationError("Actor blocked")
			}
		}
	}

	// No block means that this follower is allowed
	return nil
}

func (service *Block) FilterMessage(message *model.Message) error {

	behavior, err := service.filterMessage(message)

	if err != nil {
		return derp.Wrap(err, "service.Block.FilterMessage", "Error filtering message", message)
	}

	switch behavior {

	case model.BlockBehaviorAllow:
		message.StateID = model.InboxMessageStateReceived

	case model.BlockBehaviorMute:
		message.StateID = model.InboxMessageStateMuted

	case model.BlockBehaviorBlock:
		message.StateID = model.InboxMessageStateBlocked
	}

	return nil
}

func (service *Block) filterMessage(message *model.Message) (string, error) {

	// Get a list of all blocks for this User
	blocks, err := service.QueryByUser(message.UserID)

	if err != nil {
		return "", derp.Wrap(err, "service.Block.filterMessage", "Error loading blocks for user", message.UserID)
	}

	behavior := model.BlockBehaviorAllow

	// Try to execute each block
	for _, block := range blocks {
		if block.FilterByActorAndContent(message.Origin.URL, message.Document.Label, message.Document.Summary, message.ContentHTML) {

			switch block.Behavior {

			case model.BlockBehaviorBlock:
				return model.BlockBehaviorBlock, nil

			case model.BlockBehaviorMute:
				behavior = model.BlockBehaviorMute

			}
		}
	}

	return behavior, nil
}
