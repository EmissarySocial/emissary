package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// propertySensitive is Mastodon's ActivityPub extension that marks a post as behind a content warning
const propertySensitive = "sensitive"

// postDirectStatus creates a direct message: a note addressed only to the people it mentions,
// saved as an Object and delivered through the User's outbox instead of published as a Stream.
func postDirectStatus(factory *service.Factory, session data.Session, authorization model.Authorization, user *model.User, transaction txn.PostStatus) (object.Status, error) {

	const location = "handler.mastodon.postDirectStatus"

	// RULE: Media attaches to Streams only, so refuse it rather than silently dropping it
	if len(transaction.MediaIDs) > 0 {
		return object.Status{}, derp.BadRequest(location, "Media in direct messages is not supported yet")
	}

	inReplyTo := resolveStatusURL(factory, session, authorization, transaction.InReplyToID)
	mentions, recipients := directRecipients(factory, session, authorization, user, transaction.Status, inReplyTo)

	// RULE: A direct message with nobody to receive it would be saved and sent nowhere
	if len(recipients) == 0 {
		return object.Status{}, derp.BadRequest(location, "A direct message needs at least one @mention")
	}

	content := factory.Content().New(model.ContentFormatText, transaction.Status)
	activity := newDirectActivity(factory.Host(), user.ActivityPubURL(), content.HTML, transaction.SpoilerText, inReplyTo, mentions, recipients)

	// Save the note for its recipients only, and deliver it through the User's outbox
	if _, _, err := factory.Object().CreateAddressed(session, user.UserID, activity); err != nil {
		return object.Status{}, derp.Wrap(err, location, "Creating direct message")
	}

	status := documentToStatus(activity.Object(), user.Toot(factory.Host()))
	status.Visibility = "direct"

	return status, nil
}

// newDirectActivity builds the Create activity for a direct message: a Note whose only audience
// is its recipients, with no Public and no followers collection in its addressing.
func newDirectActivity(host string, actorURL string, contentHTML string, spoilerText string, inReplyTo string, mentions model.TagList, recipients []string) streams.Document {

	to := make([]any, 0, len(recipients))

	for _, recipient := range recipients {
		to = append(to, recipient)
	}

	tags := make([]any, 0, len(mentions))

	for _, mention := range mentions {
		tags = append(tags, mention.JSONLD(host, ""))
	}

	// Link each @mention in the text to its actor, as Mastodon does, so every reader shows them as mentions
	contentHTML = markMentionLinks(contentHTML, mentionsForDocument(streams.NewDocument(mapof.Any{vocab.PropertyTag: tags})))

	note := mapof.Any{
		vocab.PropertyType:         vocab.ObjectTypeNote,
		vocab.PropertyAttributedTo: actorURL,
		vocab.PropertyContent:      contentHTML,
		vocab.PropertyPublished:    time.Now().UTC().Format(time.RFC3339),
		vocab.PropertyTo:           to,
		vocab.PropertyTag:          tags,
	}

	if inReplyTo != "" {
		note[vocab.PropertyInReplyTo] = inReplyTo
	}

	// Mastodon shows "summary" as the content warning, and reads its own "sensitive" extension
	if spoilerText != "" {
		note[vocab.PropertySummary] = spoilerText
		note[propertySensitive] = true
	}

	return streams.NewDocument(mapof.Any{
		vocab.PropertyType:   vocab.ActivityTypeCreate,
		vocab.PropertyActor:  actorURL,
		vocab.PropertyTo:     to,
		vocab.PropertyObject: note,
	})
}

// directRecipients returns a direct message's resolved @mention tags and the actor URLs it is
// addressed to: everyone it mentions, plus the author of the post it replies to.
func directRecipients(factory *service.Factory, session data.Session, authorization model.Authorization, user *model.User, content string, inReplyTo string) (model.TagList, []string) {

	// Resolve @mentions exactly as a public post does, using a Stream that is never saved
	draft := model.NewStream()
	draft.TemplateID = "outbox-message"
	draft.Content = factory.Content().New(model.ContentFormatText, content)
	factory.Stream().CalculateMentions(&draft)

	mentions := make(model.TagList, 0, len(draft.Tags))

	for _, tag := range model.TagsOfType(draft.Tags, vocab.LinkTypeMention) {
		if tag.IsResolved() {
			mentions = append(mentions, tag)
		}
	}

	replyAuthor := replyAuthorURL(factory, session, authorization, inReplyTo)

	return mentions, collectDirectRecipients(mentions, replyAuthor, user.ActivityPubURL())
}

// collectDirectRecipients combines mentioned actors and a reply's author into one list, without
// duplicates and without the sender.
func collectDirectRecipients(mentions model.TagList, replyAuthor string, sender string) []string {

	result := make([]string, 0, len(mentions)+1)
	seen := map[string]bool{sender: true, "": true}

	add := func(actorURL string) {
		if !seen[actorURL] {
			seen[actorURL] = true
			result = append(result, actorURL)
		}
	}

	for _, tag := range mentions {
		add(tag.Href)
	}

	// RULE: A reply always reaches the author it answers, even when the text does not mention them
	add(replyAuthor)

	return result
}

// replyAuthorURL returns the actor URL of the post at postURL, falling back to the sender of the User's
// stored notification about it when the post cannot be loaded, or "" when there is no post
func replyAuthorURL(factory *service.Factory, session data.Session, authorization model.Authorization, postURL string) string {

	// The post may be unloadable when its server blocks us, or when it is a message only its recipients can read

	if postURL == "" {
		return ""
	}

	post, err := factory.ActivityStream().UserClient(authorization.UserID).Load(postURL)

	if err == nil {

		if author := post.AttributedTo().ID(); author != "" {
			return author
		}

		if author := post.ActorID(); author != "" {
			return author
		}
	}

	notifications, queryErr := factory.Notification().QueryByUserID(session, authorization.UserID, exp.Equal("objectUrl", postURL), option.MaxRows(1))

	if queryErr == nil && len(notifications) > 0 {
		return notifications[0].Actor.ProfileURL
	}

	if err != nil {
		derp.Report(derp.Wrap(err, "handler.mastodon.replyAuthorURL", "Loading replied-to post", postURL))
	}

	return ""
}
