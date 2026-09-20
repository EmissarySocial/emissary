package service

import (
	"iter"
	"net/url"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamSource manages StreamSource records, which copy content from outside of Emissary into Streams
type StreamSource struct{}

// NewStreamSource returns a fully initialized StreamSource service
func NewStreamSource() StreamSource {
	return StreamSource{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service
func (service *StreamSource) Refresh(factory *Factory) {
	// Nothing to refresh.
}

// Close stops any background processes controlled by this service
func (service *StreamSource) Close() {
	// Nothing to close.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the StreamSource collection for the provided database session
func (service *StreamSource) collection(session data.Session) data.Collection {
	return session.Collection("StreamSource")
}

// New returns a newly initialized StreamSource record
func (service *StreamSource) New() model.StreamSource {
	return model.NewStreamSource()
}

// Count returns the number of StreamSource records that match the provided criteria
func (service *StreamSource) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// List returns an iterator of the StreamSource records that match the provided criteria
func (service *StreamSource) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc over the StreamSource records that match the provided criteria
func (service *StreamSource) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.StreamSource], error) {

	iterator, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.StreamSource.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iterator, model.NewStreamSource), nil
}

// Load retrieves a StreamSource record from the database
func (service *StreamSource) Load(session data.Session, criteria exp.Expression, streamSource *model.StreamSource) error {

	if err := service.collection(session).Load(notDeleted(criteria), streamSource); err != nil {
		return derp.Wrap(err, "service.StreamSource.Load", "Loading StreamSource", criteria)
	}

	return nil
}

// Save adds or updates a StreamSource record in the database
func (service *StreamSource) Save(session data.Session, streamSource *model.StreamSource, note string) error {

	const location = "service.StreamSource.Save"

	// RULE: A StreamSource record is always attached to a Stream
	if streamSource.StreamID.IsZero() {
		return derp.BadRequest(location, "StreamSource must be attached to a Stream")
	}

	// RULE: The address is checked before the schema sees it, because the schema's own error quotes
	// the address, and an address can carry a password
	if err := validateStreamSourceURL(streamSource.URL); err != nil {
		return derp.Wrap(err, location, "Invalid URL", streamSource.StreamSourceID)
	}

	// Validate everything else against the schema
	if _, err := service.Schema().Validate(streamSource); err != nil {
		return derp.Wrap(err, location, "Validating StreamSource", streamSource.StreamSourceID)
	}

	if err := service.collection(session).Save(streamSource, note); err != nil {
		return derp.Wrap(err, location, "Saving StreamSource", streamSource.StreamSourceID, note)
	}

	return nil
}

// Delete removes a StreamSource record from the database (virtual delete)
func (service *StreamSource) Delete(session data.Session, streamSource *model.StreamSource, note string) error {

	const location = "service.StreamSource.Delete"

	// A virtual delete is safe here: the record holds no secret, because the webhook secret is derived
	if err := service.collection(session).Delete(streamSource, note); err != nil {
		return derp.Wrap(err, location, "Deleting StreamSource", streamSource.StreamSourceID, note)
	}

	// Gone, but not forgotten.  Well, forgotten in a week.
	return nil
}

// Schema returns the rosetta schema that describes a StreamSource record
func (service *StreamSource) Schema() schema.Schema {
	return schema.New(model.StreamSourceSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID retrieves a StreamSource record by its unique identifier
func (service *StreamSource) LoadByID(session data.Session, streamSourceID primitive.ObjectID, streamSource *model.StreamSource) error {

	const location = "service.StreamSource.LoadByID"

	if err := service.Load(session, exp.Equal("_id", streamSourceID), streamSource); err != nil {
		return derp.Wrap(err, location, "Loading StreamSource", streamSourceID)
	}

	return nil
}

// LoadByStreamID retrieves the StreamSource record attached to a Stream
func (service *StreamSource) LoadByStreamID(session data.Session, streamID primitive.ObjectID, streamSource *model.StreamSource) error {

	const location = "service.StreamSource.LoadByStreamID"

	if err := service.Load(session, exp.Equal("streamId", streamID), streamSource); err != nil {
		return derp.Wrap(err, location, "Loading StreamSource", streamID)
	}

	return nil
}

/******************************************
 * Status
 ******************************************/

// SetStatusLoading marks a StreamSource record as synchronizing, and saves it
func (service *StreamSource) SetStatusLoading(session data.Session, streamSource *model.StreamSource) error {

	const location = "service.StreamSource.SetStatusLoading"

	streamSource.Status = model.StreamSourceStatusLoading
	streamSource.StatusMessage = ""
	streamSource.LastSynced = time.Now().Unix()

	// Status bookkeeping skips Save's validation, which a status change can never violate
	if err := service.collection(session).Save(streamSource, "Synchronizing"); err != nil {
		return derp.Wrap(err, location, "Saving StreamSource", streamSource.StreamSourceID)
	}

	return nil
}

// SetStatusSuccess marks a StreamSource record as synchronized, and saves it
func (service *StreamSource) SetStatusSuccess(session data.Session, streamSource *model.StreamSource) error {

	const location = "service.StreamSource.SetStatusSuccess"

	streamSource.Status = model.StreamSourceStatusSuccess
	streamSource.StatusMessage = ""

	if err := service.collection(session).Save(streamSource, "Synchronized"); err != nil {
		return derp.Wrap(err, location, "Saving StreamSource", streamSource.StreamSourceID)
	}

	return nil
}

// SetStatusFailure marks a StreamSource record as failed, and saves it
func (service *StreamSource) SetStatusFailure(session data.Session, streamSource *model.StreamSource, statusMessage string) error {

	const location = "service.StreamSource.SetStatusFailure"

	streamSource.Status = model.StreamSourceStatusFailure
	streamSource.StatusMessage = truncateStatusMessage(statusMessage)

	// LastSynced is left alone, because it records when a sync last began, not when one last worked
	if err := service.collection(session).Save(streamSource, "Synchronization failed"); err != nil {
		return derp.Wrap(err, location, "Saving StreamSource", streamSource.StreamSourceID)
	}

	return nil
}

/******************************************
 * Helper Functions
 ******************************************/

// validateStreamSourceURL refuses an address that cannot be read, or that carries credentials.
// Adapters apply their own stricter rules when they read it.
func validateStreamSourceURL(value string) error {

	const location = "service.validateStreamSourceURL"

	// The parser's own error quotes the whole address, so it is replaced rather than wrapped
	parsed, err := url.Parse(value)

	if err != nil {
		return derp.BadRequest(location, "URL is not valid")
	}

	// RULE: Credentials never live in the address, where they would be stored in plain text
	if parsed.User != nil {
		return derp.BadRequest(location, "URL must not include a username or password")
	}

	return nil
}

// truncateStatusMessage shortens a status message to the length its schema allows
func truncateStatusMessage(value string) string {

	const maxLength = 1024

	if len(value) <= maxLength {
		return value
	}

	// RULE: A cut can split a multi-byte character, and MongoDB stores only valid UTF-8, so the
	// broken fragment at the end is dropped
	return strings.ToValidUTF8(value[:maxLength], "")
}
