package service

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock/metadata"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TaskFetchLinkPreview is the background task that reads one linked page's metadata
const TaskFetchLinkPreview = "FetchLinkPreview"

// How long a LinkPreview is kept before the page is read again
const (
	linkPreviewReadyLifetime  = 7 * 24 * time.Hour // a page that described itself
	linkPreviewFailedLifetime = 24 * time.Hour     // a page that could not be read, or showed nothing
)

// Longest text, in characters, that a card carries
const (
	linkPreviewMaxTitle       = 300
	linkPreviewMaxDescription = 500
)

// linkPreviewCollection names the LinkPreview collection in the common database
const linkPreviewCollection = "LinkPreview"

// linkPreviewMaxURL is the longest link, in bytes, that is read and cached
const linkPreviewMaxURL = 2000

// LinkPreview reads and caches the metadata of pages that posts link to, for their preview cards
type LinkPreview struct {
	commonDatabase func() *mongo.Database // read on every call, because a config reload can reconnect it
	queue          func() *queue.Queue    // read on every call, for the same reason
	hostname       string
	version        string

	// Tests set this to read pages on a loopback httptest server; nothing else may
	allowPrivateIPs bool
}

// NewLinkPreview returns a LinkPreview service that reads the common database and queue through the given getters
func NewLinkPreview(commonDatabase func() *mongo.Database, queue func() *queue.Queue, hostname string, version string) LinkPreview {
	return LinkPreview{
		commonDatabase: commonDatabase,
		queue:          queue,
		hostname:       hostname,
		version:        version,
	}
}

// collection returns the LinkPreview collection in the common database
func (service LinkPreview) collection() (*mongo.Collection, error) {

	const location = "service.LinkPreview.collection"

	// NILCHECK: the common database is nil while disconnected
	database := service.commonDatabase()

	if database == nil {
		return nil, derp.Internal(location, "Common database is not connected")
	}

	return database.Collection(linkPreviewCollection), nil
}

// LoadByURL returns the fresh LinkPreview for a link, or a NotFound error when there is none
func (service LinkPreview) LoadByURL(ctx context.Context, link string) (model.LinkPreview, error) {

	const location = "service.LinkPreview.LoadByURL"

	collection, err := service.collection()

	if err != nil {
		return model.LinkPreview{}, derp.Wrap(err, location, "Connecting to database")
	}

	// The TTL index deletes expired records within a minute or so, so expiry is checked here too
	result := model.LinkPreview{}
	filter := bson.M{"_id": link, "expiresAt": bson.M{"$gt": time.Now()}}

	if err := collection.FindOne(ctx, filter).Decode(&result); err != nil {

		if err == mongo.ErrNoDocuments {
			return model.LinkPreview{}, derp.NotFound(location, "No preview for this link", link)
		}

		return model.LinkPreview{}, derp.Wrap(err, location, "Loading preview", link)
	}

	return result, nil
}

// Save writes a LinkPreview, replacing any earlier one for the same link
func (service LinkPreview) Save(ctx context.Context, preview model.LinkPreview) error {

	const location = "service.LinkPreview.Save"

	collection, err := service.collection()

	if err != nil {
		return derp.Wrap(err, location, "Connecting to database")
	}

	// RULE: The link is the _id and the write is one upsert, so servers writing the same link
	// at once leave one record, whichever lands last.
	if _, err := collection.ReplaceOne(ctx, bson.M{"_id": preview.URL}, preview, options.Replace().SetUpsert(true)); err != nil {
		return derp.Wrap(err, location, "Saving preview", preview.URL)
	}

	return nil
}

// QueueFetch asks the background queue to read a link's page, once however many posts ask
func (service LinkPreview) QueueFetch(session data.Session, link string) {

	if !IsLinkPreviewURL(link) {
		return
	}

	args := mapof.Any{
		"hostname": service.hostname,
		"url":      link,
	}

	// The signature collapses every request for the same link into one queued task
	postcommit.Publish(session, service.queue(), TaskFetchLinkPreview, args, queue.WithSignature(TaskFetchLinkPreview+":"+link))
}

// Fetch reads a link's page and describes it as a LinkPreview, which is FAILED when the
// page cannot be read or has no title.  It never returns an error, because a failure is cached too.
func (service LinkPreview) Fetch(ctx context.Context, link string, now time.Time) model.LinkPreview {

	failed := model.LinkPreview{
		URL:       link,
		Status:    model.LinkPreviewStatusFailed,
		FetchedAt: now,
		ExpiresAt: now.Add(linkPreviewFailedLifetime),
	}

	// RULE: Only absolute http and https links are read
	if !IsLinkPreviewURL(link) {
		return failed
	}

	// RULE: The private-address guard is always on.  The cache is shared by every domain, so a
	// local domain's exemption would let an internal page's text become a card on a public one.
	page, err := metadata.Get(ctx, link,
		metadata.WithUserAgent(service.hostname+" /Emissary@v"+service.version+" (https://emissary.social)"),
		metadata.WithAllowPrivateIPs(service.allowPrivateIPs),
	)

	if err != nil {
		return failed
	}

	return linkPreviewFromPage(link, page, now)
}

// linkPreviewFromPage converts a page's metadata into a LinkPreview
func linkPreviewFromPage(link string, page metadata.Preview, now time.Time) model.LinkPreview {

	// A card with no title has nothing to show, so it is remembered as a failure
	title := truncateRunes(strings.TrimSpace(page.Title), linkPreviewMaxTitle)

	if title == "" {
		return model.LinkPreview{URL: link, Status: model.LinkPreviewStatusFailed, FetchedAt: now, ExpiresAt: now.Add(linkPreviewFailedLifetime)}
	}

	result := model.LinkPreview{
		URL:         link,
		Status:      model.LinkPreviewStatusReady,
		CardURL:     link,
		Title:       title,
		Description: truncateRunes(strings.TrimSpace(page.Description), linkPreviewMaxDescription),
		FetchedAt:   now,
		ExpiresAt:   now.Add(linkPreviewReadyLifetime),
	}

	// sherlock verifies that a canonical address is on the same site
	if IsLinkPreviewURL(page.URL) {
		result.CardURL = page.URL
	}

	// RULE: Every address handed to clients is http or https, never a javascript: or data: URL
	if thumbnail := page.Thumbnail; thumbnail != nil && IsLinkPreviewURL(thumbnail.URL) {
		result.Image = thumbnail.URL
		result.ImageWidth = thumbnail.Width
		result.ImageHeight = thumbnail.Height
		result.ImageDescription = truncateRunes(thumbnail.Alt, linkPreviewMaxDescription)
	}

	if provider := page.Provider; provider != nil {
		result.ProviderName = truncateRunes(provider.Name, linkPreviewMaxTitle)

		if IsLinkPreviewURL(provider.URL) {
			result.ProviderURL = provider.URL
		}
	}

	// A page that names no site is attributed to its host
	if parsed, err := url.Parse(result.CardURL); err == nil {

		if result.ProviderName == "" {
			result.ProviderName = parsed.Host
		}

		if result.ProviderURL == "" {
			result.ProviderURL = parsed.Scheme + "://" + parsed.Host
		}
	}

	if len(page.Authors) > 0 {
		result.AuthorName = truncateRunes(page.Authors[0].Name, linkPreviewMaxTitle)

		if IsLinkPreviewURL(page.Authors[0].URL) {
			result.AuthorURL = page.Authors[0].URL
		}
	}

	if page.PublishedAt != nil && !page.PublishedAt.IsZero() {
		result.PublishedAt = model.MastodonDate(*page.PublishedAt)
	}

	return result
}

// IsLinkPreviewURL returns TRUE for an absolute http or https address with a host, of a sensible length
func IsLinkPreviewURL(address string) bool {

	// RULE: The link becomes a database key and a queue signature, so an enormous one is refused
	if len(address) > linkPreviewMaxURL {
		return false
	}

	parsed, err := url.Parse(address)

	if err != nil || parsed.Host == "" {
		return false
	}

	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
