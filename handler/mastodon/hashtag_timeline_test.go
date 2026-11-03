package mastodon

import (
	"fmt"
	"testing"
	"time"

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

func TestHashtagHistory_TalliesPostsAndDistinctAccountsPerDay(t *testing.T) {

	now := time.Date(2026, 9, 26, 15, 0, 0, 0, time.UTC)

	post := func(createdAt string, account string) object.Status {
		return object.Status{CreatedAt: createdAt, Account: object.Account{URL: account}}
	}

	history := hashtagHistory([]object.Status{
		post("2026-09-26T09:00:00.000Z", "https://a.example/@alice"), // today
		post("2026-09-26T01:00:00.000Z", "https://a.example/@alice"), // today, same account
		post("2026-09-26T02:00:00.000Z", "https://b.example/@bob"),   // today, second account
		post("2026-09-24T23:59:00.000Z", "https://a.example/@alice"), // two days ago
		post("2026-09-19T12:00:00.000Z", "https://a.example/@alice"), // outside the 7 days
		post("2026-09-27T00:30:00.000Z", "https://a.example/@alice"), // in the future
		post("not a date", "https://a.example/@alice"),               // unparseable
	}, now, 7)

	require.Len(t, history, 7)

	// Newest first, so the client's "posts today" reads entry 0
	require.Equal(t, "1790380800", history[0].Day)
	require.Equal(t, "3", history[0].Uses)
	require.Equal(t, "2", history[0].Accounts)

	require.Equal(t, "0", history[1].Uses)
	require.Equal(t, "1", history[2].Uses)
	require.Equal(t, "1", history[2].Accounts)
}

func TestMergeHashtagHistories_TakesTheLargestFigurePerDay(t *testing.T) {

	histories := [][]object.TagHistory{
		{{Day: "300", Uses: "10", Accounts: "4"}, {Day: "200", Uses: "50", Accounts: "20"}},
		{{Day: "300", Uses: "25", Accounts: "3"}, {Day: "100", Uses: "7", Accounts: "2"}},
		nil, // a server that published nothing
		{{Day: "junk", Uses: "99", Accounts: "99"}},
	}

	merged := mergeHashtagHistories(histories, 2)

	// Newest first, limited to two days, one figure per day and never the sum
	require.Len(t, merged, 2)
	require.Equal(t, object.TagHistory{Day: "300", Uses: "25", Accounts: "4"}, merged[0])
	require.Equal(t, object.TagHistory{Day: "200", Uses: "50", Accounts: "20"}, merged[1])

	require.Empty(t, mergeHashtagHistories(nil, 7))
}

func TestCombineHashtagStatuses_LocalCopyWinsAndResultIsNewestFirst(t *testing.T) {

	local := []object.Status{
		{ID: "6abc", URI: "https://here.example/6abc", CreatedAt: "2026-09-26T10:00:00.000Z"},
	}

	remote := []object.Status{
		{ID: "p_copy", URI: "https://here.example/6abc", CreatedAt: "2026-09-26T10:00:00.000Z"}, // a federated copy of the local post
		{ID: "p_new", URI: "https://there.example/2", CreatedAt: "2026-09-26T12:00:00.000Z"},
		{ID: "p_old", URI: "https://there.example/1", CreatedAt: "2026-09-25T09:00:00.000Z"},
	}

	combined := combineHashtagStatuses(local, remote, 2, false)

	// One copy of the shared post, keeping the local ID; newest first; cut to the limit
	require.Len(t, combined, 2)
	require.Equal(t, "p_new", combined[0].ID)
	require.Equal(t, "6abc", combined[1].ID)
}
