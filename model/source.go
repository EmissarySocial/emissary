package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SourceAdapter enumerates all of the possible values for a stream.Source variable
type SourceAdapter string

// SourceAdapterActivityPub identifies a Stream that originated on an external ActivityPub server
const SourceAdapterActivityPub SourceAdapter = "ACTIVITYPUB"

// SourceAdapterEmail identifies a Stream that originated on an external Email server
const SourceAdapterEmail SourceAdapter = "EMAIL"

// SourceAdapterRSS identifies a Stream that originated on an external RSS feed
const SourceAdapterRSS SourceAdapter = "RSS"

// SourceAdapterSystem identifies a Stream that originated on this server
const SourceAdapterSystem SourceAdapter = "SYSTEM"

// SourceAdapterTwitter identifies a Stream that originated on Twitter
const SourceAdapterTwitter SourceAdapter = "TWITTER"

// SourceMethod enumerates the different kind of data sources
type SourceMethod string

// SourceMethodPoll identifies that this source must be polled to provide data
const SourceMethodPoll SourceMethod = "POLL"

// SourceMethodWebhook identifies that this source supports webhooks
const SourceMethodWebhook SourceMethod = "WEBHOOK"

// SourceConfig stores all configuration data about a specific remote stream source
type SourceConfig map[string]string

// Source represents an account or node on this server.
type Source struct {
	SourceID primitive.ObjectID `json:"sourceId" bson:"_id"`    // This is the internal ID for the domain.  It should not be available via the web service.
	Label    string             `json:"label"    bson:"label"`  // Fully qualified domain name (without protocol)
	Adapter  SourceAdapter      `json:"type"     bson:"type"`   // What kind of source
	Method   SourceMethod       `json:"method"   bson:"method"` // How do we connect to the source? Polling or WebHooks?
	Config   SourceConfig       `jwsn:"config"   bson:"config"` // Source-specific configuration information.  This is validated by JSON-Schema provided by the source adapter.

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the unique identifier of this object, and fulfills part of the data.Object interface
func (source *Source) ID() string {
	return source.SourceID.Hex()
}
