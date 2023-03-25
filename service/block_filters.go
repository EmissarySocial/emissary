package service

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Filters
 ******************************************/

func (service *Block) FilterMessage(message *model.Message) error {

	behavior, err := service.filter(message.UserID, message.Origin, message.Document, message.ContentHTML)

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

func (service *Block) filter(userID primitive.ObjectID, origin model.OriginLink, document model.DocumentLink, contentHTML string) (string, error) {

	blocks, err := service.QueryByUser(userID)

	if err != nil {
		return "", derp.Wrap(err, "service.Block.Filter", "Error loading blocks for user", userID.Hex())
	}

	// Reset the message state to "received"
	result := model.BlockBehaviorAllow

	// Try to apply each block filter to the message
	for _, block := range blocks {

		var err error
		var behavior string

		switch block.Type {

		case model.BlockTypeActor:
			behavior = filter_Actor(block, document)

		case model.BlockTypeDomain:
			behavior = filter_Domain(block, document)

		case model.BlockTypeContent:
			behavior = filter_Content(block, document, contentHTML)

		case model.BlockTypeExternal:
			behavior, err = filter_External(block, document, contentHTML)
		}

		// Handle all errors
		if err != nil {
			return "", derp.Wrap(err, "service.Block.Filter", "Error filtering message", block, document, contentHTML)
		}

		switch behavior {

		// "Block" is the highest priority action, so we can return immediately
		case model.BlockBehaviorBlock:
			return model.BlockBehaviorBlock, nil

		// If we get a "Mute", then save that value, but keep looking for a "Block"
		case model.BlockBehaviorMute:
			result = model.BlockBehaviorMute

		// Anything else (like "Allow") will not change the starting result
		// and should not override any "Mute" that has already been set
		default:
		}
	}

	return result, nil
}

func filter_Actor(block model.Block, document model.DocumentLink) string {

	behavior := model.BlockBehaviorAllow

	// Verify each actor in the AttributedTo list
	for _, actor := range document.AttributedTo {

		// If this actor is blocked, then collect the behavior to return
		// TODO: Assume that other servers are hostile, so block matching should be more "fuzzy" than this.
		// Strip out protocols, aliases, etc. Can we use Sherlock to retrieve more information about an Actor.
		if (block.Trigger == actor.ProfileURL) || (block.Trigger == actor.EmailAddress) {

			switch block.Behavior {

			case model.BlockBehaviorBlock:
				return model.BlockBehaviorBlock

			case model.BlockBehaviorMute:
				behavior = model.BlockBehaviorMute
			}
		}
	}

	return behavior
}

func filter_Domain(block model.Block, document model.DocumentLink) string {

	behavior := model.BlockBehaviorAllow

	// Collect all domains that we need to check
	domains := make([]string, 1, len(document.AttributedTo)+1)

	domains[0] = document.URL
	for _, actor := range document.AttributedTo {
		domains = append(domains, actor.ProfileURL)
	}

	// Check all domains
	for _, domain := range domains {
		if domainURL, err := url.Parse(domain); err == nil {
			if strings.HasSuffix(domainURL.Host, block.Trigger) {
				switch block.Behavior {

				case model.BlockBehaviorBlock:
					return model.BlockBehaviorBlock

				case model.BlockBehaviorMute:
					behavior = model.BlockBehaviorMute
				}
			}
		}
	}

	for _, actor := range document.AttributedTo {
		if strings.HasSuffix(actor.EmailAddress, block.Trigger) {
			switch block.Behavior {

			case model.BlockBehaviorBlock:
				return model.BlockBehaviorBlock

			case model.BlockBehaviorMute:
				behavior = model.BlockBehaviorMute
			}
		}
	}

	return behavior
}

func filter_Content(block model.Block, document model.DocumentLink, contentHTML string) string {

	result := model.BlockBehaviorAllow

	scannable := []*string{
		&document.Label,
		&document.Summary,
		&contentHTML,
	}

	for _, text := range scannable {
		if strings.Contains(*text, block.Trigger) {
			switch block.Behavior {

			case model.BlockBehaviorBlock:
				return model.BlockBehaviorBlock

			case model.BlockBehaviorMute:
				result = model.BlockBehaviorMute
			}
		}
	}

	return result
}

func filter_External(block model.Block, document model.DocumentLink, contentHTML string) (string, error) {
	return "", nil
}
