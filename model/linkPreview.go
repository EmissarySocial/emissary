package model

import (
	"time"

	"github.com/benpate/toot/object"
)

// LinkPreview statuses
const (
	LinkPreviewStatusReady  = "READY"  // the page described itself well enough to draw a card
	LinkPreviewStatusFailed = "FAILED" // the page could not be fetched, or had nothing to show
)

// LinkPreview is the cached description of one web page, read from its own metadata,
// that becomes the preview card on any post linking to it.  It is shared by every domain.
type LinkPreview struct {
	URL              string    `bson:"_id"`                        // The link exactly as it appears in posts
	Status           string    `bson:"status"`                     // READY or FAILED
	CardURL          string    `bson:"cardUrl,omitempty"`          // The page's canonical address
	Title            string    `bson:"title,omitempty"`            // The page's title
	Description      string    `bson:"description,omitempty"`      // The page's summary
	Image            string    `bson:"image,omitempty"`            // The page's preview picture
	ImageWidth       int       `bson:"imageWidth,omitempty"`       // Width of the preview picture, in pixels
	ImageHeight      int       `bson:"imageHeight,omitempty"`      // Height of the preview picture, in pixels
	ImageDescription string    `bson:"imageDescription,omitempty"` // Alt text for the preview picture
	ProviderName     string    `bson:"providerName,omitempty"`     // Name of the site that publishes the page
	ProviderURL      string    `bson:"providerUrl,omitempty"`      // Home page of that site
	AuthorName       string    `bson:"authorName,omitempty"`       // Name of the page's author
	AuthorURL        string    `bson:"authorUrl,omitempty"`        // Profile of the page's author
	PublishedAt      string    `bson:"publishedAt,omitempty"`      // When the page was published, in Mastodon's date format
	FetchedAt        time.Time `bson:"fetchedAt"`                  // When the page was read
	ExpiresAt        time.Time `bson:"expiresAt"`                  // When this record is deleted and the page may be read again
}

// IsFresh returns TRUE if this record may still be used at the given time
func (preview LinkPreview) IsFresh(now time.Time) bool {
	return now.Before(preview.ExpiresAt)
}

// Card returns this page as a Mastodon preview card, or nil when it has nothing to show
func (preview LinkPreview) Card() *object.PreviewCard {

	if preview.Status != LinkPreviewStatusReady {
		return nil
	}

	result := &object.PreviewCard{
		URL:              preview.CardURL,
		Title:            preview.Title,
		Description:      preview.Description,
		Type:             "link",
		AuthorName:       preview.AuthorName,
		AuthorURL:        preview.AuthorURL,
		ProviderName:     preview.ProviderName,
		ProviderURL:      preview.ProviderURL,
		Image:            preview.Image,
		Width:            preview.ImageWidth,
		Height:           preview.ImageHeight,
		ImageDescription: preview.ImageDescription,
		PublishedAt:      preview.PublishedAt,
		Authors:          []object.PreviewCardAuthor{},
	}

	if result.URL == "" {
		result.URL = preview.URL
	}

	if preview.AuthorName != "" {
		result.Authors = append(result.Authors, object.PreviewCardAuthor{Name: preview.AuthorName, URL: preview.AuthorURL})
	}

	return result
}
