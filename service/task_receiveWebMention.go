package service

import (
	"bytes"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskReceiveWebMention struct {
	streamService  *Stream
	mentionService *Mention
	userService    *User
	source         string
	target         string
}

func NewTaskReceiveWebMention(streamService *Stream, mentionService *Mention, userService *User, source string, target string) TaskReceiveWebMention {
	return TaskReceiveWebMention{
		streamService:  streamService,
		mentionService: mentionService,
		userService:    userService,
		source:         source,
		target:         target,
	}
}

func (task TaskReceiveWebMention) Run() error {

	const location = "service.TaskReceiveWebMention.Run"

	var content bytes.Buffer

	// Validate that the WebMention source actually links to the targetURL
	if err := task.mentionService.Verify(task.source, task.target, &content); err != nil {
		return derp.Wrap(err, location, "Source does not link to target", task.source, task.target)
	}

	// Parse the target URL into an object type and token
	objectType, token, err := task.mentionService.ParseURL(task.target)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing URL", task.target)
	}

	var objectID primitive.ObjectID

	// Validate the internal record that the mention is pointing to
	switch objectType {

	case model.MentionTypeStream:
		stream := model.NewStream()
		if err := task.streamService.LoadByToken(token, &stream); err != nil {
			return derp.Wrap(err, location, "Cannot load stream", task.target)
		}
		objectID = stream.StreamID

	case model.MentionTypeUser:
		user := model.NewUser()
		if err := task.userService.LoadByToken(token, &user); err != nil {
			return derp.Wrap(err, location, "Cannot load user", token)
		}
		objectID = user.UserID

	default:
		return derp.NewInternalError(location, "Unknown Mention Type.  This should never happen", objectType)
	}

	// Check the database for an existing Mention record
	mention, err := task.mentionService.LoadOrCreate(objectType, objectID, task.source)

	if err != nil {
		return derp.Wrap(err, location, "Error loading mention", objectType, token)
	}

	// Parse the WebMention source into the Mention object
	if err := task.mentionService.GetPageInfo(&content, task.source, &mention); err != nil {
		return derp.Wrap(err, location, "Error parsing source", task.source)
	}

	// Try to save the mention to the database
	if err := task.mentionService.Save(&mention, "Created"); err != nil {
		return derp.Wrap(err, location, "Error saving mention")
	}

	return nil
}
