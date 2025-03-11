package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
)

// saveToInbox adds/updates an individual Message based on an RSS item.  It returns TRUE if a new record was created
func (service *Following) SaveMessage(following *model.Following, document streams.Document, originType string) error {

	const location = "service.Following.SaveMessage"

	// If collapseThreads is set, then traverse "inReplyTo" values back to the primary document
	if following.CollapseThreads {
		document, originType = getPrimaryPost(document, originType)
	}

	// Convert the document into a message (and traverse responses if necessary)
	message := getMessage(following, document, originType)

	// Try to save a unique version of this message to the database (always collapse duplicates)
	if err := service.saveUniqueMessage(message); err != nil {
		return derp.Wrap(err, location, "Error saving message")
	}

	// Yee. Haw.
	return nil
}

// saveUnique adds/updates a message in the database.  If the message.URL does not already
// exist, then a new message is added to the Inbox.  Otherwise, the "references" data will
// of the existing record be updated and the unique value will be re-saved.
func (service *Following) saveUniqueMessage(message model.Message) error {

	const location = "service.Following.saveUnique"

	// Search for a previous UNREAD message with our same UserID and URL.
	previousMessage := model.Message{}

	if err := service.inboxService.LoadByURL(message.UserID, message.URL, &previousMessage); !derp.NilOrNotFound(err) {
		return derp.Wrap(err, location, "Error searching for duplicate message", message)
	}

	// If no previous message was found, then save the current message as is
	if previousMessage.IsNew() {

		if err := service.inboxService.Save(&message, "Created"); err != nil {
			return derp.Wrap(err, location, "Error saving new message", message)
		}

		return nil
	}

	// Fall through means that we have a duplicate message.

	// Try to update the previousMessage with a new origin (a new reply, like, etc)
	isReferenceUpdated := previousMessage.AddReference(message.Origin)
	isStatusUpdated := false

	// Update the message status to "NEW-REPLIES" so that previously
	// read messages will show up again in the Inbox.
	if message.Origin.Type == model.OriginTypeReply {
		isStatusUpdated = previousMessage.MarkNewReplies()
	}

	// if the message was updated (from AddReference or MarkNewReplies) then save it.
	if isReferenceUpdated || isStatusUpdated {
		if err := service.inboxService.Save(&previousMessage, "Message Imported"); err != nil {
			return derp.Wrap(err, location, "Error updating previous message with new origin and status", previousMessage)
		}
	}

	// Successfully updated the message, or not.  But still, it's good.
	return nil
}

/******************************************
 * Helper Functions
 ******************************************/

// getPrimaryPost traverses UP a chain of replies to locate the first message that was posted.
// If there are one or more replies in the chain, then the returned originType is "REPLY"
// TODO: LOW: In the future, the "context" value may be useful in traversing this list.
func getPrimaryPost(document streams.Document, originType string) (streams.Document, string) {

	// Unwrap "activity" documents
	document = document.UnwrapActivity()

	if inReplyTo := document.InReplyTo(); inReplyTo.NotNil() {

		// Change origin type from PRIMARY to REPLY without affecting
		// LIKE and DISLIKE types
		if originType == model.OriginTypePrimary {
			originType = model.OriginTypeReply
		}

		// Traverse up the tree.  If the "primary" document is found, then return that instead.
		if primaryDocument, originType := getPrimaryPost(inReplyTo.LoadLink(), originType); primaryDocument.NotNil() {
			return primaryDocument, originType
		}
	}

	return document, originType
}

// getMessage returns an inbox Message object based on the provided arguments.
func getMessage(following *model.Following, document streams.Document, originType string) model.Message {

	result := model.NewMessage()
	result.UserID = following.UserID
	result.FollowingID = following.FollowingID
	result.FolderID = following.FolderID
	result.SocialRole = document.Type()
	result.URL = document.ID()
	result.InReplyTo = document.InReplyTo().ID()
	result.PublishDate = document.Published().Unix()
	result.AddReference(following.Origin(originType))

	return result
}
