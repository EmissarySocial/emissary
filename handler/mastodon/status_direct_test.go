package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// mentionTag returns a resolved @mention of the provided actor
func mentionTag(actorURL string) model.Tag {
	return model.Tag{Type: vocab.LinkTypeMention, Name: "@someone", Href: actorURL}
}

// TestCollectDirectRecipients_MentionsAndReplyAuthor confirms every mentioned actor and the
// replied-to author are addressed, once each and in order
func TestCollectDirectRecipients_MentionsAndReplyAuthor(t *testing.T) {

	mentions := model.TagList{mentionTag("https://a.example/@alice"), mentionTag("https://b.example/@bob")}

	result := collectDirectRecipients(mentions, "https://c.example/@carol", "https://me.example/@me")

	require.Equal(t, []string{"https://a.example/@alice", "https://b.example/@bob", "https://c.example/@carol"}, result)
}

// TestCollectDirectRecipients_ReplyAuthorAlreadyMentioned confirms an author who is also
// mentioned is addressed only once
func TestCollectDirectRecipients_ReplyAuthorAlreadyMentioned(t *testing.T) {

	mentions := model.TagList{mentionTag("https://a.example/@alice"), mentionTag("https://a.example/@alice")}

	result := collectDirectRecipients(mentions, "https://a.example/@alice", "https://me.example/@me")

	require.Equal(t, []string{"https://a.example/@alice"}, result)
}

// TestCollectDirectRecipients_NeverTheSender confirms the sender is never one of the recipients,
// even when mentioning themselves or replying to their own post
func TestCollectDirectRecipients_NeverTheSender(t *testing.T) {

	const me = "https://me.example/@me"

	result := collectDirectRecipients(model.TagList{mentionTag(me)}, me, me)

	require.Empty(t, result)
}

// TestCollectDirectRecipients_NoReply confirms a message that is not a reply is addressed to its
// mentions alone
func TestCollectDirectRecipients_NoReply(t *testing.T) {

	result := collectDirectRecipients(model.TagList{mentionTag("https://a.example/@alice")}, "", "https://me.example/@me")

	require.Equal(t, []string{"https://a.example/@alice"}, result)
}

// TestNewDirectActivity_AddressesOnlyRecipients confirms the note and its activity go to the
// recipients alone, never to Public or a followers collection
func TestNewDirectActivity_AddressesOnlyRecipients(t *testing.T) {

	recipients := []string{"https://a.example/@alice", "https://b.example/@bob"}
	mentions := model.TagList{mentionTag("https://a.example/@alice")}

	activity := newDirectActivity("https://me.example", "https://me.example/@me", "<p>hi</p>", "", "https://b.example/@bob/1", mentions, recipients)

	require.Equal(t, vocab.ActivityTypeCreate, activity.Type())
	require.Equal(t, "https://me.example/@me", activity.ActorID())
	require.False(t, activity.IsPublic())
	require.ElementsMatch(t, recipients, activity.Recipients())

	note := activity.Object()
	require.Equal(t, vocab.ObjectTypeNote, note.Type())
	require.False(t, note.IsPublic())
	require.ElementsMatch(t, recipients, note.Recipients())
	require.Equal(t, "https://b.example/@bob/1", note.InReplyTo().ID())
	require.Equal(t, "<p>hi</p>", note.Content())
	require.Equal(t, "", note.Summary())
}

// TestNewDirectActivity_ContentWarning confirms spoiler text becomes Mastodon's content warning
func TestNewDirectActivity_ContentWarning(t *testing.T) {

	activity := newDirectActivity("https://me.example", "https://me.example/@me", "<p>hi</p>", "spoilers", "", nil, []string{"https://a.example/@alice"})

	note := activity.Object()
	require.Equal(t, "spoilers", note.Summary())
	require.True(t, note.Get(propertySensitive).Bool())
	require.True(t, note.InReplyTo().IsNil())
}

// TestNewDirectActivity_StatusIsDirect confirms a Mastodon client reads the note back as a direct message
func TestNewDirectActivity_StatusIsDirect(t *testing.T) {

	activity := newDirectActivity("https://me.example", "https://me.example/@me", "<p>hi</p>", "", "", nil, []string{"https://a.example/@alice"})

	require.Equal(t, "direct", documentVisibility(activity.Object()))
}

// TestNewDirectActivity_LinksMentions confirms each @mention in the text becomes a link to its actor
func TestNewDirectActivity_LinksMentions(t *testing.T) {

	alice := model.Tag{Type: vocab.LinkTypeMention, Name: "alice@a.example", Href: "https://a.example/@alice"}

	activity := newDirectActivity("https://me.example", "https://me.example/@me", "<p>@alice@a.example hi</p>", "", "", model.TagList{alice}, []string{"https://a.example/@alice"})

	content := activity.Object().Content()
	require.Contains(t, content, `href="https://a.example/@alice"`)
	require.Contains(t, content, "hi")
}
