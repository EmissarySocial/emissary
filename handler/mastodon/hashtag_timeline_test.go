package mastodon

import (
	"fmt"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

func remoteStatus(uri string, createdAt string) object.Status {
	return object.Status{
		ID:         "109999",
		URI:        uri,
		CreatedAt:  createdAt,
		Content:    "<p>hello <a href=\"https://example.com/tags/cats\">#cats</a></p>",
		Favourited: true,
		Bookmarked: true,
		Account: object.Account{
			ID:       "42",
			Acct:     "alice",
			Username: "alice",
			URL:      "https://example.com/@alice",
			Note:     "<p>bio</p>",
		},
	}
}

func TestRemapRemoteStatus_ReplacesIDsAndClearsViewerState(t *testing.T) {

	status := remoteStatus("https://example.com/users/alice/statuses/1", "2026-09-18T09:00:00.000Z")
	status.InReplyToID = "555"
	status.InReplyToAccountID = "77"
	status.Poll = &object.Poll{ID: "9"}
	status.Mentions = []object.StatusMention{{ID: "8", Username: "bob", Acct: "bob", URL: "https://example.com/@bob"}}

	result, ok := remapRemoteStatus(status)

	require.True(t, ok)
	require.Equal(t, model.EncodeRemoteStatusID("https://example.com/users/alice/statuses/1"), result.ID)
	require.Equal(t, model.EncodeRemoteAccountID("https://example.com/@alice"), result.Account.ID)
	require.Equal(t, "alice@example.com", result.Account.Acct)
	require.Equal(t, "bob@example.com", result.Mentions[0].Acct)
	require.Equal(t, model.EncodeRemoteAccountID("https://example.com/@bob"), result.Mentions[0].ID)

	// The remote's own IDs and viewer state mean nothing here
	require.Empty(t, result.InReplyToID)
	require.Empty(t, result.InReplyToAccountID)
	require.Nil(t, result.Poll)
	require.False(t, result.Favourited)
	require.False(t, result.Bookmarked)
}

func TestRemapRemoteStatus_SanitizesHTML(t *testing.T) {

	status := remoteStatus("https://example.com/users/alice/statuses/1", "2026-09-18T09:00:00.000Z")
	status.Content = `<p>hi</p><script>alert(1)</script><img src=x onerror="alert(2)">`
	status.Account.Note = `<p>bio</p><script>alert(3)</script>`

	result, ok := remapRemoteStatus(status)

	require.True(t, ok)
	require.NotContains(t, result.Content, "<script")
	require.NotContains(t, result.Content, "onerror")
	require.NotContains(t, result.Account.Note, "<script")
	require.Contains(t, result.Content, "hi")
}

func TestRemapRemoteStatus_KeepsAnAlreadyQualifiedAcct(t *testing.T) {

	status := remoteStatus("https://example.com/users/alice/statuses/1", "2026-09-18T09:00:00.000Z")
	status.Account.Acct = "alice@example.com"

	result, _ := remapRemoteStatus(status)
	require.Equal(t, "alice@example.com", result.Account.Acct)
}

func TestRemapRemoteStatus_RejectsUnusableStatuses(t *testing.T) {

	boost := remoteStatus("https://example.com/users/alice/statuses/2", "2026-09-18T09:00:00.000Z")
	boost.Reblog = &object.Status{}

	noURI := remoteStatus("", "2026-09-18T09:00:00.000Z")

	noAccountURL := remoteStatus("https://example.com/users/alice/statuses/3", "2026-09-18T09:00:00.000Z")
	noAccountURL.Account.URL = ""

	for name, status := range map[string]object.Status{"boost": boost, "no uri": noURI, "no account url": noAccountURL} {
		_, ok := remapRemoteStatus(status)
		require.False(t, ok, name)
	}
}

func TestMergeHashtagStatuses_DedupesSortsAndLimits(t *testing.T) {

	a := []object.Status{
		remoteStatus("https://a.example/1", "2026-09-18T09:00:00.000Z"),
		remoteStatus("https://shared.example/9", "2026-09-18T11:00:00.000Z"),
	}
	b := []object.Status{
		remoteStatus("https://shared.example/9", "2026-09-18T11:00:00.000Z"), // same post, seen from a second server
		remoteStatus("https://b.example/2", "2026-09-18T10:00:00.000Z"),
	}

	result := mergeHashtagStatuses([][]object.Status{a, nil, b}, 10, false)

	require.Len(t, result, 3)
	require.Equal(t, "https://shared.example/9", result[0].URI)
	require.Equal(t, "https://b.example/2", result[1].URI)
	require.Equal(t, "https://a.example/1", result[2].URI)

	require.Len(t, mergeHashtagStatuses([][]object.Status{a, b}, 2, false), 2)
}

func TestMergeHashtagStatuses_OnlyMedia(t *testing.T) {

	withMedia := remoteStatus("https://a.example/1", "2026-09-18T09:00:00.000Z")
	withMedia.MediaAttachments = []object.MediaAttachment{sizedMedia("image", 640, 480)}
	plain := remoteStatus("https://a.example/2", "2026-09-18T10:00:00.000Z")

	result := mergeHashtagStatuses([][]object.Status{{withMedia, plain}}, 10, true)

	require.Len(t, result, 1)
	require.Equal(t, "https://a.example/1", result[0].URI)
}

func TestHostsOf(t *testing.T) {

	urls := []string{
		"https://mastodon.social/users/Gargron",
		"https://mastodon.social/users/other", // duplicate host
		"https://own.example/@me",             // this server itself
		"not a url ::",
		"",
		"https://fosstodon.org/users/fosstodon",
	}

	require.Equal(t, []string{"mastodon.social", "fosstodon.org"}, hostsOf(urls, "own.example"))
}

func TestHostsOf_CapsTheFanOut(t *testing.T) {

	urls := make([]string, 0, 30)

	for i := range 30 {
		urls = append(urls, fmt.Sprintf("https://host%d.example/users/a", i))
	}

	require.Len(t, hostsOf(urls, "own.example"), hashtagMaxHosts)
}

func sizedMedia(mediaType string, width float64, height float64) object.MediaAttachment {
	return object.MediaAttachment{
		ID:   "1",
		Type: mediaType,
		URL:  "https://a.example/file",
		Meta: map[string]any{"original": map[string]any{"width": width, "height": height}},
	}
}

func TestSafeMediaAttachments(t *testing.T) {

	unsized := object.MediaAttachment{ID: "2", Type: "image", URL: "https://a.example/nosize.png", Meta: map[string]any{}}
	noMeta := object.MediaAttachment{ID: "3", Type: "video", URL: "https://a.example/nometa.mp4"}
	unknown := sizedMedia("unknown", 100, 100)
	zero := sizedMedia("image", 0, 480)
	audio := object.MediaAttachment{ID: "4", Type: "audio", URL: "https://a.example/song.mp3"}
	noURL := sizedMedia("image", 100, 100)
	noURL.URL = ""

	result := safeMediaAttachments([]object.MediaAttachment{
		sizedMedia("image", 640, 480),
		sizedMedia("gifv", 320, 240),
		unsized, noMeta, unknown, zero, noURL, audio,
	})

	require.Len(t, result, 3)
	require.Equal(t, "image", result[0].Type)
	require.Equal(t, "gifv", result[1].Type)
	require.Equal(t, "audio", result[2].Type)
}

// A post whose only media is unusable must come back with no media at all,
// never with an attachment the client can't size.
func TestRemapRemoteStatus_DropsUnsizedMedia(t *testing.T) {

	status := remoteStatus("https://example.com/users/alice/statuses/1", "2026-09-18T09:00:00.000Z")
	status.MediaAttachments = []object.MediaAttachment{{ID: "1", Type: "unknown", URL: "https://example.com/x"}}

	result, ok := remapRemoteStatus(status)

	require.True(t, ok)
	require.Empty(t, result.MediaAttachments)
}
