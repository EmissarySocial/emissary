package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/sherlock/metadata"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// linkPreviewPage is a page that describes itself with Open Graph tags
const linkPreviewPage = `<!doctype html><html><head>
<title>Fallback title</title>
<meta property="og:title" content="A Good Story">
<meta property="og:description" content="What happened, in brief.">
<meta property="og:image" content="https://cdn.example/story.jpg">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:site_name" content="The Example Times">
</head><body>Story</body></html>`

// newLinkPageServer serves linkPreviewPage on a loopback address
func newLinkPageServer(t *testing.T) *httptest.Server {

	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(linkPreviewPage))
	}))

	t.Cleanup(server.Close)
	return server
}

// TestLinkPreview_Fetch_RefusesPrivateAddresses confirms no server reads a page on its own network,
// a local one included, because the cache it writes is shared with every domain
func TestLinkPreview_Fetch_RefusesPrivateAddresses(t *testing.T) {

	page := newLinkPageServer(t)
	now := time.Now()

	for _, hostname := range []string{"social.example", "localhost"} {

		refused := NewLinkPreview(nil, nil, hostname, "0.9.0").Fetch(context.Background(), page.URL, now)

		require.Equal(t, model.LinkPreviewStatusFailed, refused.Status, hostname)
		require.Nil(t, refused.Card(), hostname)
		require.Equal(t, now.Add(linkPreviewFailedLifetime), refused.ExpiresAt, hostname)
	}
}

// TestLinkPreview_Fetch_ReadsPage confirms a page's own metadata becomes the preview
func TestLinkPreview_Fetch_ReadsPage(t *testing.T) {

	page := newLinkPageServer(t)
	now := time.Now()

	// The test page is on loopback, which only a test may read
	local := NewLinkPreview(nil, nil, "social.example", "0.9.0")
	local.allowPrivateIPs = true
	read := local.Fetch(context.Background(), page.URL, now)

	require.Equal(t, model.LinkPreviewStatusReady, read.Status)
	require.Equal(t, "A Good Story", read.Title)
	require.Equal(t, "https://cdn.example/story.jpg", read.Image)
	require.Equal(t, 1200, read.ImageWidth)
	require.Equal(t, "The Example Times", read.ProviderName)
	require.Equal(t, now.Add(linkPreviewReadyLifetime), read.ExpiresAt)
}

// TestLinkPreview_Fetch_RefusesNonWebLinks confirms only http and https links are read
func TestLinkPreview_Fetch_RefusesNonWebLinks(t *testing.T) {

	service := NewLinkPreview(nil, nil, "social.example", "0.9.0")
	service.allowPrivateIPs = true

	tooLong := "https://news.example/" + strings.Repeat("a", linkPreviewMaxURL)

	for _, link := range []string{"", "/relative", "file:///etc/passwd", "javascript:alert(1)", "ftp://files.example/x", tooLong} {
		require.Equal(t, model.LinkPreviewStatusFailed, service.Fetch(context.Background(), link, time.Now()).Status, link)
	}
}

// TestLinkPreviewFromPage confirms what a page's metadata contributes to a card, and what it may not
func TestLinkPreviewFromPage(t *testing.T) {

	now := time.Now()
	published := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

	page := metadata.Preview{
		URL:         "https://news.example/story",
		Title:       "  " + strings.Repeat("t", linkPreviewMaxTitle+50) + "  ",
		Description: "Summary",
		Thumbnail:   &metadata.Thumbnail{URL: "javascript:alert(1)"},
		Provider:    &metadata.Provider{Name: "", URL: "data:text/html,x"},
		Authors:     []metadata.Author{{Name: "Pat", URL: "https://news.example/pat"}},
		PublishedAt: &published,
	}

	result := linkPreviewFromPage("https://news.example/story?ref=feed", page, now)

	require.Equal(t, model.LinkPreviewStatusReady, result.Status)
	require.Len(t, []rune(result.Title), linkPreviewMaxTitle, "long titles are cut")
	require.Equal(t, "https://news.example/story", result.CardURL, "the canonical address is used")
	require.Empty(t, result.Image, "an image that is not http or https is dropped")
	require.Equal(t, "news.example", result.ProviderName, "a page that names no site is attributed to its host")
	require.Equal(t, "https://news.example", result.ProviderURL, "a provider address that is not http or https is replaced")
	require.Equal(t, "Pat", result.AuthorName)
	require.Equal(t, model.MastodonDate(published), result.PublishedAt)

	// A page with no title has nothing to show
	untitled := linkPreviewFromPage("https://news.example/blank", metadata.Preview{Description: "only this"}, now)
	require.Equal(t, model.LinkPreviewStatusFailed, untitled.Status)
}

// TestLinkPreview_SaveAndLoad runs the cache against a real MongoDB: one record per link, and expiry honored
func TestLinkPreview_SaveAndLoad(t *testing.T) {

	database := newDomainTestDatabase(t)
	service := NewLinkPreview(func() *mongo.Database { return database }, func() *queue.Queue { return nil }, "social.example", "0.9.0")
	ctx := context.Background()
	now := time.Now()

	// Nothing cached yet
	_, err := service.LoadByURL(ctx, "https://news.example/story")
	require.True(t, derp.IsNotFound(err))

	// Two servers saving the same link leave one record, the last one written
	first := model.LinkPreview{URL: "https://news.example/story", Status: model.LinkPreviewStatusFailed, FetchedAt: now, ExpiresAt: now.Add(time.Hour)}
	second := model.LinkPreview{URL: "https://news.example/story", Status: model.LinkPreviewStatusReady, Title: "A Good Story", FetchedAt: now, ExpiresAt: now.Add(time.Hour)}

	require.NoError(t, service.Save(ctx, first))
	require.NoError(t, service.Save(ctx, second))

	count, err := database.Collection(linkPreviewCollection).CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	loaded, err := service.LoadByURL(ctx, "https://news.example/story")
	require.NoError(t, err)
	require.Equal(t, "A Good Story", loaded.Card().Title)

	// An expired record reads as missing, before MongoDB's TTL monitor deletes it
	expired := model.LinkPreview{URL: "https://news.example/old", Status: model.LinkPreviewStatusReady, Title: "Old", FetchedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour)}
	require.NoError(t, service.Save(ctx, expired))

	_, err = service.LoadByURL(ctx, "https://news.example/old")
	require.True(t, derp.IsNotFound(err))
}

// TestLinkPreview_DisconnectedDatabase confirms a missing common database is an error, not a panic
func TestLinkPreview_DisconnectedDatabase(t *testing.T) {

	service := NewLinkPreview(func() *mongo.Database { return nil }, func() *queue.Queue { return nil }, "social.example", "0.9.0")

	_, err := service.LoadByURL(context.Background(), "https://news.example/story")
	require.Error(t, err)
	require.False(t, derp.IsNotFound(err), "a disconnected database must not read as an uncached link")

	require.Error(t, service.Save(context.Background(), model.LinkPreview{URL: "https://news.example/story"}))
}
