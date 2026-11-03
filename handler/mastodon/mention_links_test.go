package mastodon

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// testMentions are two accounts a post can mention, one on another server and one local-style.
var testMentions = []object.StatusMention{
	{ID: "u_x", Username: "devInTheVoid", URL: "https://mastodon.social/ap/users/1", Acct: "devInTheVoid@mastodon.social"},
	{ID: "abc", Username: "bob", URL: "https://example.com/@bob", Acct: "bob@example.com"},
}

// devLink is the markup expected for the devInTheVoid mention.
const devLink = `<span class="h-card"><a href="https://mastodon.social/ap/users/1" class="u-url mention">@<span>devInTheVoid</span></a></span>`

// bobLink is the markup expected for the bob mention.
const bobLink = `<span class="h-card"><a href="https://example.com/@bob" class="u-url mention">@<span>bob</span></a></span>`

// TestMarkMentionLinks confirms mentions become Mastodon-style links showing the short name, and only whole mentions in plain text
func TestMarkMentionLinks(t *testing.T) {

	require.Equal(t, devLink+" hello", markMentionLinks("@devInTheVoid@mastodon.social hello", testMentions))
	require.Equal(t, "hi "+devLink+"!", markMentionLinks("hi @devInTheVoid@mastodon.social!", testMentions), "punctuation after a mention stays outside the link")
	require.Equal(t, devLink+" and "+bobLink, markMentionLinks("@devInTheVoid@mastodon.social and @bob@example.com", testMentions), "several mentions in one post")
	require.Equal(t, "<p>"+devLink+" and "+devLink+"</p>", markMentionLinks("<p>@devinthevoid@MASTODON.social and @devInTheVoid@mastodon.social</p>", testMentions), "matching ignores case, inside markup too")

	// Not a mention of that account
	require.Equal(t, "mail me: x@devInTheVoid@mastodon.social", markMentionLinks("mail me: x@devInTheVoid@mastodon.social", testMentions), "the tail of a longer address")
	require.Equal(t, "@devInTheVoid@mastodon.social.evil.com", markMentionLinks("@devInTheVoid@mastodon.social.evil.com", testMentions), "a longer domain is a different account")
	require.Equal(t, "@someoneElse@mastodon.social", markMentionLinks("@someoneElse@mastodon.social", testMentions), "only accounts in the mentions list")

	// A sentence ending in a mention is still a mention
	require.Equal(t, "thanks "+devLink+".", markMentionLinks("thanks @devInTheVoid@mastodon.social.", testMentions))

	// Already a link: left as it is
	existing := `<a href="https://mastodon.social/ap/users/1">@devInTheVoid@mastodon.social</a>`
	require.Equal(t, existing, markMentionLinks(existing, testMentions))

	// Nothing to do
	require.Equal(t, "plain text", markMentionLinks("plain text", testMentions))
	require.Equal(t, "@devInTheVoid@mastodon.social", markMentionLinks("@devInTheVoid@mastodon.social", nil))
	require.Equal(t, "", markMentionLinks("", testMentions))

	// Text that needs escaping stays safe
	require.Equal(t, "1 &lt; 2 "+devLink+" &amp; more", markMentionLinks("1 &lt; 2 @devInTheVoid@mastodon.social &amp; more", testMentions))
}

// TestMarkMentionLinks_CaseChangesByteLength pins mentions after text whose lowercase form has a
// different length in bytes. Offsets from a lowercased copy once overran the text and panicked.
func TestMarkMentionLinks_CaseChangesByteLength(t *testing.T) {

	// "Ⱥ" is 2 bytes and lowercases to the 3-byte "ⱥ", so 20 of them push an offset past the end
	grows := strings.Repeat("Ⱥ", 20)
	require.Equal(t, grows+" "+bobLink, markMentionLinks(grows+" @bob@example.com", testMentions))
	require.Equal(t, grows+" "+bobLink+" "+grows, markMentionLinks(grows+" @bob@example.com "+grows, testMentions))

	// The 3-byte Kelvin sign lowercases to a 1-byte "k", which once cut the text mid-character
	require.Equal(t, "K "+bobLink, markMentionLinks("K @bob@example.com", testMentions))
	require.Equal(t, "KKK "+bobLink+" KKK", markMentionLinks("KKK @bob@example.com KKK", testMentions))

	// The mention itself can change length when its case does
	alice := []object.StatusMention{{ID: "alice", Username: "ⱥlice", URL: "https://example.com/@alice", Acct: "ⱥlice@example.com"}}
	aliceLink := `<span class="h-card"><a href="https://example.com/@alice" class="u-url mention">@<span>ⱥlice</span></a></span>`
	require.Equal(t, "hi "+aliceLink+"!", markMentionLinks("hi @ȺLICE@example.com!", alice))
}

// TestMarkMentionLinks_InvalidUTF8 pins that malformed text never panics. Lowercasing turned each
// invalid byte into a 3-byte replacement character, which also overran the text.
func TestMarkMentionLinks_InvalidUTF8(t *testing.T) {

	require.NotPanics(t, func() {
		result := markMentionLinks(strings.Repeat("\xff", 20)+" @bob@example.com", testMentions)
		require.Contains(t, result, bobLink)
	})

	require.NotPanics(t, func() {
		markMentionLinks("@bob@example.com\xff\xfe", testMentions)
	})

	// An invalid byte in an address matches only the same byte, never a different invalid one
	broken := []object.StatusMention{{ID: "x", Username: "x", URL: "https://example.com/@x", Acct: "x\xff@example.com"}}
	require.Equal(t, "@x\xfe@example.com", markMentionLinks("@x\xfe@example.com", broken))
}

// FuzzMarkMentionLinks asserts that no post content or mention address can panic the mention
// linker, and that content with nothing to link comes back unchanged.
func FuzzMarkMentionLinks(f *testing.F) {

	f.Add("@devInTheVoid@mastodon.social hello", "bob@example.com")
	f.Add(strings.Repeat("Ⱥ", 20)+" @bob@example.com", "bob@example.com")
	f.Add("K @bob@example.com", "bob@example.com")
	f.Add("hi @ȺLICE@example.com!", "ⱥlice@example.com")
	f.Add("\xff\xfe @bob@example.com", "bob@example.com")
	f.Add("<p>@bob@example.com</p><a>@bob@example.com</a>", "bob@example.com")
	f.Add("@", "")

	f.Fuzz(func(t *testing.T, content string, acct string) {

		mentions := []object.StatusMention{{ID: "x", Username: "x", URL: "https://example.com/@x", Acct: acct}}

		// Every input must come back without a panic
		result := markMentionLinks(content, mentions)

		// Content with an "@" may gain links; content without one has nothing to link
		if strings.Contains(content, "@") {
			return
		}

		require.Equal(t, content, result)
	})
}

// TestMentionsForStream confirms a post's Mention tags become the API's mentions list
func TestMentionsForStream(t *testing.T) {

	userID := primitive.NewObjectID()

	stream := model.NewStream()
	stream.URL = "https://example.com/" + stream.StreamID.Hex()
	stream.Tags = model.TagList{
		{Type: vocab.LinkTypeHashtag, Name: "cat"},
		{Type: vocab.LinkTypeMention, Name: "devInTheVoid@mastodon.social", Href: "https://mastodon.social/ap/users/1"},
		{Type: vocab.LinkTypeMention, Name: "me", Href: "https://example.com/@" + userID.Hex()},
		{Type: vocab.LinkTypeMention, Name: "nohref@example.org"},
	}

	none := func(string) string { return "" }

	mentions := mentionsForStream(&stream, none)
	require.Len(t, mentions, 2, "hashtags and mentions without a profile URL are left out")

	require.Equal(t, "devInTheVoid", mentions[0].Username)
	require.Equal(t, "devInTheVoid@mastodon.social", mentions[0].Acct)
	require.Equal(t, model.EncodeRemoteAccountID("https://mastodon.social/ap/users/1"), mentions[0].ID, "a remote account gets the encoded ID")

	require.Equal(t, "me", mentions[1].Username)
	require.Equal(t, userID.Hex(), mentions[1].ID, "a User on this server gets their own ID")
}

// TestMentionsForStream_Unresolved confirms a mention whose lookup failed (stored as "-") is never
// used as an address, and is completed from known records when they have it.
func TestMentionsForStream_Unresolved(t *testing.T) {

	stream := model.NewStream()
	stream.URL = "https://example.com/" + stream.StreamID.Hex()
	stream.Tags = model.TagList{
		{Type: vocab.LinkTypeMention, Name: "devInTheVoid@mastodon.social", Href: model.TagHrefUnresolvable},
		{Type: vocab.LinkTypeMention, Name: "stranger@elsewhere.net", Href: model.TagHrefUnresolvable},
		{Type: vocab.LinkTypeMention, Name: "fresh@new.example"},
	}

	known := func(acct string) string {
		if acct == "devInTheVoid@mastodon.social" {
			return "https://mastodon.social/ap/users/117111409165181205"
		}
		return ""
	}

	mentions := mentionsForStream(&stream, known)
	require.Len(t, mentions, 1, "unknown people with no usable address are left out, and a dash is never an address")
	require.Equal(t, "https://mastodon.social/ap/users/117111409165181205", mentions[0].URL)
	require.Equal(t, model.EncodeRemoteAccountID("https://mastodon.social/ap/users/117111409165181205"), mentions[0].ID)

	nobody := mentionsForStream(&stream, func(string) string { return "" })
	require.Empty(t, nobody, "a dash is not a profile URL")
}

// TestMentionsForDocument_UsesTheLinkWrittenInTheContent covers a remote post whose mention tag points
// at the actor URL while its content links the profile page; the tapped link is the profile page.
func TestMentionsForDocument_UsesTheLinkWrittenInTheContent(t *testing.T) {

	document := streams.NewDocument(map[string]any{
		"id":      "https://fosstodon.org/users/davep/statuses/1",
		"type":    "Note",
		"content": `<p><span><a href="https://fosstodon.org/@fosstodon" rel="nofollow">@<span>fosstodon</span></a></span> hello</p>`,
		"tag": []any{
			map[string]any{"type": "Mention", "href": "https://fosstodon.org/users/fosstodon", "name": "@fosstodon"},
			map[string]any{"type": "Hashtag", "href": "https://fosstodon.org/tags/go", "name": "#go"},
		},
	})

	mentions := mentionsForDocument(document)

	require.Len(t, mentions, 1)
	require.Equal(t, "https://fosstodon.org/@fosstodon", mentions[0].URL)
	require.Equal(t, "fosstodon", mentions[0].Username)
	require.Equal(t, "fosstodon@fosstodon.org", mentions[0].Acct)
	require.Equal(t, model.EncodeRemoteAccountID("https://fosstodon.org/users/fosstodon"), mentions[0].ID)
}

// TestMentionsForDocument_FallsBackToTheActorURL covers a mention that has no link in the content.
func TestMentionsForDocument_FallsBackToTheActorURL(t *testing.T) {

	document := streams.NewDocument(map[string]any{
		"id":      "https://example.com/p/1",
		"type":    "Note",
		"content": "<p>hi @someone@other.net</p>",
		"tag":     []any{map[string]any{"type": "Mention", "href": "https://other.net/users/someone", "name": "@someone@other.net"}},
	})

	mentions := mentionsForDocument(document)

	require.Len(t, mentions, 1)
	require.Equal(t, "https://other.net/users/someone", mentions[0].URL)
	require.Equal(t, "someone@other.net", mentions[0].Acct)
}
