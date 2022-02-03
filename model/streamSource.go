package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

////////////////////////////

// StreamSourceAdapter enumerates all of the possible values for a stream.StreamSource variable
type StreamSourceAdapter string

func (sourceAdapter StreamSourceAdapter) String() string {
	return string(sourceAdapter)
}

// StreamSourceAdapterActivityPub identifies a Stream that originated on an external ActivityPub server
const StreamSourceAdapterActivityPub StreamSourceAdapter = "ACTIVITYPUB"

// StreamSourceAdapterEmail identifies a Stream that originated on an external Email server
const StreamSourceAdapterEmail StreamSourceAdapter = "EMAIL"

// StreamSourceAdapterRSS identifies a Stream that originated on an external RSS feed
const StreamSourceAdapterRSS StreamSourceAdapter = "RSS"

// StreamSourceAdapterSystem identifies a Stream that originated on this server
const StreamSourceAdapterSystem StreamSourceAdapter = "SYSTEM"

// StreamSourceAdapterTwitter identifies a Stream that originated on Twitter
const StreamSourceAdapterTwitter StreamSourceAdapter = "TWITTER"

// StreamSourceMethod enumerates the different kind of data sources
type StreamSourceMethod string

func (sourceMethod StreamSourceMethod) String() string {
	return string(sourceMethod)
}

// StreamSourceMethodPoll identifies that this source must be polled to provide data
const StreamSourceMethodPoll StreamSourceMethod = "POLL"

// StreamSourceMethodWebhook identifies that this source supports webhooks
const StreamSourceMethodWebhook StreamSourceMethod = "WEBHOOK"

////////////////////////

// StreamSourceConfig stores all configuration data about a specific remote stream source
type StreamSourceConfig map[string]string

// StreamSource represents an account or node on this server.
type StreamSource struct {
	StreamSourceID primitive.ObjectID  `json:"streamSourceId" bson:"_id"`     // This is the internal ID for the domain.  It should not be available via the web service.
	Label          string              `json:"label"          bson:"label"`   // Fully qualified domain name (without protocol)
	Adapter        StreamSourceAdapter `json:"adapter"        bson:"adapter"` // What kind of source
	Method         StreamSourceMethod  `json:"method"         bson:"method"`  // How do we connect to the source? Polling or WebHooks?
	Config         StreamSourceConfig  `jwsn:"config"         bson:"config"`  // StreamSource-specific configuration information.  This is validated by JSON-Schema provided by the source adapter.

	journal.Journal `json:"journal" bson:"journal"`
}

/*******************************************
 * DATA.OBJECT INTERFACE
 *******************************************/

// ID returns the primary key of this object
func (source *StreamSource) ID() string {
	return source.StreamSourceID.Hex()
}
