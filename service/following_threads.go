package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/sherlock"
)

// saveToInbox adds/updates an individual Message based on an RSS item.  It returns TRUE if a new record was created
func (service *Following) saveMessage(following model.Following, document streams.Document) error {

	const location = "service.Following.saveMessage"

	// Traverse JSON-LD documents if necessary
	document = getActualDocument(document)

	// Load and refine the document from its actual URL
	document, _ = service.activityStreams.Load(document.ID(), sherlock.WithDefaultValue(document.Map()))

	// Convert the document into a message (and traverse responses if necessary)
	message, err := service.getMessage(document, following.CollapseThreads)

	if err != nil {
		return derp.Wrap(err, location, "Error converting document to message")
	}

	// Set additional values for the message based on the following record
	message.UserID = following.UserID
	message.FolderID = following.FolderID
	message.Origin = service.getOrigin(following, document)

	// Try to save a unique version of this message to the database (always collapse duplicates)
	if err := service.saveUnique(message); err != nil {
		return derp.Wrap(err, location, "Error saving message")
	}

	// Yee. Haw.
	return nil
}

// getOrigin returns an OriginLink object based on the provided document.  If the document is a
// traversable link (Like, Dislike, Announce, or Reply) then the OriginLink will include that
// information.
func (service *Following) getOrigin(following model.Following, document streams.Document) model.OriginLink {

	result := following.Origin()

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

// getMessage returns a Message object based on the provides args.  If following.collapseThreads is TRUE,
// then this function will follow replies, boosts, and likes to their original source, and return that instead.
// If following.collapseThreads is FALSE, then this document will be converted into a message directly.
func (service *Following) getMessage(document streams.Document, collapse bool) (model.Message, error) {

	// Try to load the document from the Interwebs
	document, err := document.Load()

	if err != nil {
		return model.Message{}, derp.Wrap(err, "service.Following.getMessage", "Error loading document")
	}

	// If we're told to collapse threads, then look for traversable links
	// (Likes, Dislikes, Announces, and Replies) and try to follow them
	// to their source
	if collapse {

		nextDocument := streams.NilDocument()

		switch document.Type() {

		case vocab.ActivityTypeLike:
			nextDocument = document.Object()

		case vocab.ActivityTypeDislike:
			nextDocument = document.Object()

		case vocab.ActivityTypeAnnounce:
			nextDocument = document.Object()

		default:
			if inReplyTo := document.InReplyTo(); inReplyTo.NotNil() {
				nextDocument = inReplyTo
			}
		}

		// If we have a traversable link, then try to follow it to the source.
		if nextDocument.NotNil() {
			if result, err := service.getMessage(nextDocument, collapse); err == nil {
				return result, nil
			}
		}
	}

	// Fall through means that there are no traversable links (or we got
	// an error trying to resolve one). So this document is as good as it's
	// going to get...

	result := model.NewMessage()
	result.URL = document.ID()
	result.Label = document.Name()
	result.Summary = document.Summary()
	result.ImageURL = document.Image().URL()
	result.AttributedTo = convert.ActivityPubPersonLink(document.AttributedTo())
	result.ContentHTML = document.Content()
	result.PublishDate = document.Published().Unix()

	return result, nil
}

// saveUnique adds/updates a message in the database.  If the message.URL does not already
// exist, then a new message is added to the Inbox.  Otherwise, the "references" data will
// of the existing record be updated and the unique value will be re-saved.
func (service *Following) saveUnique(message model.Message) error {

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
