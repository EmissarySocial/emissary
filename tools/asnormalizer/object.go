package asnormalizer

import (
	"strings"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Object normalizes a regular document (Article, Note, etc)
func Object(client streams.Client, document streams.Document) map[string]any {

	const location = "tools.asnormalizer.Object"

	actual := document.UnwrapActivity()

	// This function is for Articles, Notes, etc.
	// If the actual document is not an object then
	// it must be normalized by someone else.
	if actual.NotObject() {
		return nil
	}

	actual = unwrapEmptyPages(actual)

	actorID := first(actual.Actor().ID(), document.Actor().ID())

	result := map[string]any{
		vocab.PropertyType:         actual.Type(),
		vocab.PropertyID:           actual.ID(),
		vocab.PropertyActor:        actorID,
		vocab.PropertyAttributedTo: first(actual.AttributedTo().ID(), actorID),
		vocab.PropertyInReplyTo:    actual.InReplyTo().ID(),
		vocab.PropertyReplies:      actual.Replies().ID(),
		vocab.PropertyName:         actual.Name(),
		vocab.PropertyContext:      Context(document),
		vocab.PropertySummary:      actual.Summary(),
		vocab.PropertyContent:      actual.Content(),
		vocab.PropertyPublished:    first(actual.Published(), time.Now()),
		vocab.PropertyTag:          Tags(document.Tag()),
	}

	// Expand the "AttributedTo" actor
	if attributedToID := actual.AttributedTo().ID(); attributedToID != "" {

		attributedTo, err := client.Load(attributedToID)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to load attributedTo actor", attributedToID))
		}

		result[vocab.PropertyAttributedTo] = ActorSummary(attributedTo)
	}

	// Normalize Attachments
	if attachments := actual.Attachment(); attachments.NotNil() {

		for attachment := attachments; attachment.NotNil(); attachment = attachment.Tail() {
			if strings.HasPrefix(attachment.MediaType(), "image") {
				result[vocab.PropertyImage] = AttachmentAsImage(attachment)
				break
			}
		}

		result[vocab.PropertyAttachment] = Attachment(attachments)
	}

	if image := actual.Image(); image.NotNil() {
		result[vocab.PropertyImage] = Image(image)
	}

	if icon := actual.Icon(); icon.NotNil() {
		result[vocab.PropertyIcon] = Image(icon)
	}

	return result
}

func unwrapEmptyPages(activity streams.Document) streams.Document {

	if activity.Type() != vocab.ObjectTypePage {
		return activity
	}

	if activity.Content() != "" {
		return activity
	}

	object := activity.Object()

	if object.IsNil() {
		return activity
	}

	if object.Type() != vocab.ObjectTypeNote {
		return activity
	}

	return object
}
