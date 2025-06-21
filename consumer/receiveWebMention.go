package consumer

import (
	"bytes"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ReceiveWebMention(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.ReceiveWebMention"

	// Collect Arguments
	mentionService := factory.Mention()
	source := args.GetString("source")
	target := args.GetString("target")

	// Validate that the WebMention source actually links to the targetURL
	var content bytes.Buffer
	if err := mentionService.Verify(source, target, &content); err != nil {
		return queue.Error(derp.Wrap(err, location, "Source does not link to target", source, target))
	}

	// Parse the target URL into an object type and token
	objectType, token, err := mentionService.ParseURL(target)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error parsing URL", target))
	}

	var objectID primitive.ObjectID

	// Validate the internal record that the mention is pointing to
	switch objectType {

	case model.MentionTypeStream:
		streamService := factory.Stream()
		stream := model.NewStream()
		if err := streamService.LoadByToken(token, &stream); err != nil {
			return queue.Error(derp.Wrap(err, location, "Cannot load stream", target))
		}
		objectID = stream.StreamID

	case model.MentionTypeUser:
		userService := factory.User()
		user := model.NewUser()
		if err := userService.LoadByToken(token, &user); err != nil {
			return queue.Error(derp.Wrap(err, location, "Cannot load user", token))
		}
		objectID = user.UserID

	default:
		return queue.Error(derp.InternalError(location, "Unknown Mention Type.  This should never happen", objectType))
	}

	// Check the database for an existing Mention record
	mention, err := mentionService.LoadOrCreate(objectType, objectID, source)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error loading mention", objectType, token))
	}

	// Parse the WebMention source into the Mention object
	if err := mentionService.GetPageInfo(&content, source, &mention); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error parsing source", source))
	}

	// Try to save the mention to the database
	if err := mentionService.Save(&mention, "Created"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error saving mention"))
	}

	return queue.Success()
}
