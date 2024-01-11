package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// saveToInbox adds/updates an individual Message based on an RSS item.  It returns TRUE if a new record was created
func (service *Following) SaveMessage(following *model.Following, document streams.Document) error {

	const location = "service.Following.SaveMessage"

	// Convert the document into a message (and traverse responses if necessary)
	message, err := service.getMessage(following, document)

	if err != nil {
		return derp.Wrap(err, location, "Error converting document to message")
	}

	// Set the origin based on the original document (not the object of the message)
	message.AddReference(service.getOrigin(following, document))

	// Try to save a unique version of this message to the database (always collapse duplicates)
	if err := service.saveUniqueMessage(message); err != nil {
		return derp.Wrap(err, location, "Error saving message")
	}

	// Yee. Haw.
	return nil
}

// getMessage returns a Message object based on the provides args.  If following.collapseThreads is TRUE,
// then this function will follow replies, boosts, and likes to their original source, and return that instead.
// If following.collapseThreads is FALSE, then this document will be converted into a message directly.
func (service *Following) getMessage(following *model.Following, document streams.Document) (model.Message, error) {

	nextDocument := streams.NilDocument()

	switch document.Type() {

	// Always follow Likes
	case vocab.ActivityTypeLike:
		nextDocument = document.Object()

	// Always follow Dislikes
	case vocab.ActivityTypeDislike:
		nextDocument = document.Object()

	// Always follow Announces
	case vocab.ActivityTypeAnnounce:
		nextDocument = document.Object()

	// Follow InReplyTo links if we're collapsing threads
	default:
		if following.CollapseThreads {
			if inReplyTo := document.InReplyTo(); inReplyTo.NotNil() {
				nextDocument = inReplyTo
			}
		}
	}

	// If we have a traversable link, then try to follow it to the source.
	if nextDocument.NotNil() {
		if result, err := service.getMessage(following, nextDocument); err == nil {
			return result, nil
		}
	}

	// Fall through means that there are no more traversable links (or we got
	// an error trying to resolve one). So this document is as good as we're
	// going to get.  Make a new message to return to the caller.

	result := model.NewMessage()
	result.UserID = following.UserID
	result.FolderID = following.FolderID
	result.AddReference(service.getOrigin(following, document))
	result.SocialRole = document.Type()
	result.URL = document.ID()
	// result.Label = document.Name()
	// result.Summary = document.Summary()
	// result.ImageURL = document.Image().URL()
	// result.AttributedTo = convert.ActivityPubAttributedTo(document)
	// result.ContentHTML = document.Content()
	result.InReplyTo = document.InReplyTo().ID()

	if publishDate := document.Published().Unix(); publishDate > 0 {
		result.PublishDate = publishDate
	} else if updateDate := document.Updated().Unix(); updateDate > 0 {
		result.PublishDate = updateDate
	} else {
		result.PublishDate = time.Now().Unix()
	}

	return result, nil
}

// saveUnique adds/updates a message in the database.  If the message.URL does not already
// exist, then a new message is added to the Inbox.  Otherwise, the "references" data will
// of the existing record be updated and the unique value will be re-saved.
func (service *Following) saveUniqueMessage(message model.Message) error {

	const location = "service.Following.saveUnique"

	// Search for a previous UNREAD message with our same UserID and URL.
	// RULE: If we're adding onto a previously read message, then it's OK to duplicate the URL.
	previousMessage := model.Message{}

	if err := service.inboxService.LoadUnreadByURL(message.UserID, message.URL, &previousMessage); err != nil {

		// Report legitimate errors to the authorities
		if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error searching for message")
		}

		// If a previous message doesn't exist, then we can save the new directly
		if err := service.inboxService.Save(&message, "Message Imported"); err != nil {
			return derp.Wrap(err, location, "Error saving message")
		}

		return nil
	}

	// Fall through means we have a duplicate, so try to add the message origin to the previous message and save.
	if updated := previousMessage.AddReference(message.Origin); updated {

		if err := service.inboxService.Save(&message, "Additional References Added"); err != nil {
			return derp.Wrap(err, location, "Error saving message")
		}
	}

	return nil
}

// getOrigin returns an OriginLink object based on the provided document.  If the document is a
// traversable link (Like, Dislike, Announce, or Reply) then the OriginLink will include that
// information.
func (service *Following) getOrigin(following *model.Following, document streams.Document) model.OriginLink {

	result := following.Origin()

	// Try to add a little more information about how we got here: Like, Dislike, Announce, Reply
	switch document.Type() {

	case vocab.ActivityTypeLike:
		result.Type = model.OriginTypeLike

	case vocab.ActivityTypeDislike:
		result.Type = model.OriginTypeDislike

	case vocab.ActivityTypeAnnounce:
		result.Type = model.OriginTypeAnnounce

	default:
		if inReplyTo := document.InReplyTo(); inReplyTo.NotNil() {
			result.Type = model.OriginTypeReply
		}
	}

	return result
}
