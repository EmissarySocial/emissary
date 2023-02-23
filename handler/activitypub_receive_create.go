package handler

import (
	"encoding/json"
	"time"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeCreate, vocab.Any, func(factory *domain.Factory, user *model.User, document streams.Document) error {

		object, err := document.Object().AsObject()

		if err != nil {
			return derp.Wrap(err, "handler.activitypub_receive_create", "Error getting object from document")
		}

		// RULE: Ignore these Object Types
		switch object.Type() {
		case vocab.ObjectTypeRelationship,
			vocab.ObjectTypeProfile,
			vocab.ObjectTypePlace,
			vocab.ObjectTypeEvent,
			vocab.ObjectTypeTombstone:
			return nil
		}

		// Try to validate the "Following" object
		// TODO: How would this work for private or unsolicited messages?
		followingService := factory.Following()
		following := model.NewFollowing()
		if err := followingService.LoadByURL(user.UserID, document.ActorID(), &following); err != nil {
			return derp.Wrap(err, "handler.activitypub_receive_create", "Error loading following record", user.UserID, object.ActorID())
		}

		message := model.NewMessage()
		message.UserID = user.UserID
		message.Origin = following.Origin()
		message.Document = model.DocumentLink{
			Type:       object.Type(),
			URL:        object.ID(),
			Label:      object.Name(),
			Summary:    object.Summary(),
			ImageURL:   object.ImageURL(),
			UpdateDate: time.Now().Unix(),
		}

		if author := object.AttributedTo(); !author.IsNil() {
			if author, err := author.AsObject(); err == nil {
				message.Document.Author = model.PersonLink{
					Name:       author.Name(),
					ProfileURL: author.ID(),
					ImageURL:   author.ImageURL(),
				}
			}
		}

		message.ContentHTML = object.Content()
		message.FolderID = following.FolderID

		if publishDate := document.Published().Unix(); publishDate > 0 {
			message.PublishDate = publishDate
		} else if updateDate := document.Updated().Unix(); updateDate > 0 {
			message.PublishDate = updateDate
		} else {
			message.PublishDate = time.Now().Unix()
		}

		if contentJSON, err := json.Marshal(document); err == nil {
			message.ContentJSON = string(contentJSON)
		}

		inboxService := factory.Inbox()

		// OMG, is that it? Are we done?  Let's see....
		if err := inboxService.Save(&message, "Created via ActivityPub"); err != nil {
			return derp.Wrap(err, "handler.activitypub_receive_create", "Error saving message", user.UserID, message.Document.URL)
		}

		// Hooo-dat?!?!?
		return nil
	})
}
