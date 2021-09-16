package reader

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/benpate/activitystream/vocabulary"
	"github.com/benpate/derp"
)

// Object is a data structure that makes it easy(-er) to read and understand data from an ActivityStream JSON package.
// In the process, it makes irreversible changes to the original data set, so it MUST NOT be used to generate new ActivityStream
// records.  Use ObjectWriter for that purpose.
type Object struct {
	currentRecord map[string]interface{}
	nextRecords   []interface{}
}

// New converts the provided input into a fully initialized Object reference, or returns an error trying.
func New(input interface{}) (*Object, error) {

	object := Object{
		currentRecord: map[string]interface{}{},
		nextRecords:   []interface{}{},
	}

	switch input := input.(type) {

	case map[string]interface{}:
		object.currentRecord = input

	case string:
		if err := json.Unmarshal([]byte(input), &object.currentRecord); err != nil {
			return nil, derp.New(http.StatusBadRequest, "activitystream.New", "Unable to unmarshal JSON", err, input)
		}

	default:
		return nil, derp.New(http.StatusBadRequest, "activitystream.New", "Unrecognized input type", input)
	}

	return &object, nil
}

func NewFromMap(input map[string]interface{}) *Object {

	return &Object{
		currentRecord: input,
		nextRecords:   []interface{}{},
	}
}

// ID provides the globally unique identifier for an Object or Link
func (object Object) ID() string {
	return object.ParseString(vocabulary.PropertyID, "")
}

// Type identifies the Object or Link type.  If multiple values are present, then only the first value is returned.
func (object Object) Type() string {
	return object.ParseString(vocabulary.PropertyType, "")
}

// TypeArray lists out all types in the configured object
func (object Object) TypeArray() []string {
	return object.ParseStringArray(vocabulary.PropertyType)
}

// Actor describes one or more entities that either performed or are expected to perform the activity. Any single activity can have multiple actors. The actor MAY be specified using an indirect Link.
func (object Object) Actor() string {
	return object.ParseString(vocabulary.PropertyActor, vocabulary.PropertyID)
}

// ActorObject returns a fully object for the "Actor" property
func (object Object) ActorObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyActor, vocabulary.PropertyID)
}

// Attachment identifies a resource attached or related to an object that potentially requires special handling. The intent is to provide a model that is at least semantically similar to attachments in email.
func (object Object) Attachment() string {
	return object.ParseString(vocabulary.PropertyAttachment, vocabulary.PropertyURL)
}

// AttachmentObject returns a fully populated object for the "attachment" property
func (object Object) AttachmentObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyAttachment, vocabulary.PropertyURL)
}

// AttributedTo identifies one or more entities to which this object is attributed. The attributed entities might not be Actors. For instance, an object might be attributed to the completion of another activity.
func (object Object) AttributedTo() string {
	return object.ParseString(vocabulary.PropertyAttributedTo, vocabulary.PropertyID)
}

// AttributedToObject returns a fully populated object for the "attributedTo" property
func (object Object) AttributedToObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyAttributedTo, vocabulary.PropertyID)
}

// Audience identifies one or more entities that represent the total population of entities for which the object can considered to be relevant.
func (object Object) Audience() string {
	return object.ParseString(vocabulary.PropertyAudience, vocabulary.PropertyID)
}

// AudienceObject returns a fully object for the "Audience" property
func (object Object) AudienceObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyAudience, vocabulary.PropertyID)
}

// Bcc identifies one or more Objects that are part of the private secondary audience of this Object.
func (object Object) Bcc() string {
	return object.ParseString(vocabulary.PropertyBcc, vocabulary.PropertyID)
}

// BccObject returns a fully object for the "Bcc" property
func (object Object) BccObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyBcc, vocabulary.PropertyID)
}

// BTo identifies an Object that is part of the private primary audience of this Object.
func (object Object) BTo() string {
	return object.ParseString(vocabulary.PropertyBTo, vocabulary.PropertyID)
}

// BToObject returns a fully object for the "BTo" property
func (object Object) BToObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyBTo, vocabulary.PropertyID)
}

// Cc identifies an Object that is part of the public secondary audience of this Object.
func (object Object) Cc() string {
	return object.ParseString(vocabulary.PropertyCc, vocabulary.PropertyID)
}

// CcObject returns a fully object for the "Cc" property
func (object Object) CcObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyCc, vocabulary.PropertyID)
}

// Closed indicates that a question has been closed, and answers are no longer accepted.
func (object Object) Closed() time.Time {
	return object.ParseTime(vocabulary.PropertyClosed)
}

// ClosedObject returns a fully object for the "Closed" property
func (object Object) ClosedObject() time.Time {
	return object.ParseTime(vocabulary.PropertyClosed)
}

// Content is the content or textual representation of the Object encoded as a JSON string. By default, the value of content is HTML. The mediaType property can be used in the object to indicate a different content type.  The content MAY be expressed using multiple language-tagged values.
func (object Object) Content(language ...string) string {
	return object.ParseMap(vocabulary.PropertyContent, language...)
}

// Context Identifies the context within which the object exists or an activity was performed.  The notion of "context" used is intentionally vague. The intended function is to serve as a means of grouping objects and activities that share a common originating context or purpose. An example could be all activities relating to a common project or event.
func (object Object) Context() (Context, bool) {
	return object.ParseContext()
}

// Duration - when the object describes a time-bound resource, such as an audio or video, a meeting, etc, the duration property indicates the object's approximate duration. The value MUST be expressed as an xsd:duration as defined by [ xmlschema11-2], section 3.3.6 (e.g. a period of 5 seconds is represented as "PT5S").
func (object Object) Duration() time.Duration {
	return object.ParseDuration(vocabulary.PropertyDuration)
}

// DurationObject returns a fully object for the "Duration" property
func (object Object) DurationObject() time.Duration {
	return object.ParseDuration(vocabulary.PropertyDuration)
}

// EndTime represents the date and time describing the actual or expected ending time of the object. When used with an Activity object, for instance, the endTime property specifies the moment the activity concluded or is expected to conclude.
func (object Object) EndTime() time.Time {
	return object.ParseTime(vocabulary.PropertyEndTime)
}

// EndTimeObject returns a fully object for the "EndTime" property
func (object Object) EndTimeObject() time.Time {
	return object.ParseTime(vocabulary.PropertyEndTime)
}

// Generator identifies the entity (e.g. an application) that generated the object.
func (object Object) Generator() string {
	return object.ParseString(vocabulary.PropertyGenerator, vocabulary.PropertyID)
}

// GeneratorObject returns a fully object for the "Generator" property
func (object Object) GeneratorObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyGenerator, vocabulary.PropertyID)
}

// Icon indicates an entity that describes an icon for this object. The image should have an aspect ratio of one (horizontal) to one (vertical) and should be suitable for presentation at a small size.
func (object Object) Icon() string {
	return object.ParseString(vocabulary.PropertyIcon, vocabulary.PropertyID)
}

// IconObject returns a fully object for the "Icon" property
func (object Object) IconObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyIcon, vocabulary.PropertyID)
}

// Image indicates an entity that describes an image for this object. Unlike the icon property, there are no aspect ratio or display size limitations assumed.
func (object Object) Image() string {
	return object.ParseString(vocabulary.PropertyImage, vocabulary.PropertyID)
}

// ImageObject returns a fully object for the "Image" property
func (object Object) ImageObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyImage, vocabulary.PropertyID)
}

// InReplyTo indicates one or more entities for which this object is considered a response.
func (object Object) InReplyTo() string {
	return object.ParseString(vocabulary.PropertyInReplyTo, vocabulary.PropertyID)
}

// InReplyToObject returns a fully object for the "InReplyTo" property
func (object Object) InReplyToObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyInReplyTo, vocabulary.PropertyID)
}

// Instrument identifies one or more objects used (or to be used) in the completion of an Activity.
func (object Object) Instrument() string {
	return object.ParseString(vocabulary.PropertyInstrument, vocabulary.PropertyID)
}

// InstrumentObject returns a fully object for the "instrument" property
func (object Object) InstrumentObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyInstrument, vocabulary.PropertyID)
}

// Location indicates one or more physical or logical locations associated with the object.
func (object Object) Location() string {
	return object.ParseString(vocabulary.PropertyLocation, vocabulary.PropertyID)
}

// LocationObject returns a fully object for the "Location" property
func (object Object) LocationObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyLocation, vocabulary.PropertyID)
}

// Name is a simple, human-readable, plain-text name for the object. HTML markup MUST NOT be included. The name MAY be expressed using multiple language-tagged values.
func (object Object) Name(language ...string) string {
	return object.ParseMap(vocabulary.PropertyName, language...)
}

// Origin describes an indirect object of the activity from which the activity is directed. The precise meaning of the origin is the object of the English preposition "from". For instance, in the activity "John moved an item to List B from List A", the origin of the activity is "List A".
func (object Object) Origin() string {
	return object.ParseString(vocabulary.PropertyOrigin, vocabulary.PropertyID)
}

// OriginObject returns a fully object for the "Origin" property
func (object Object) OriginObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyOrigin, vocabulary.PropertyID)
}

// Preview identifies an entity that provides a preview of this object.
func (object Object) Preview() string {
	return object.ParseString(vocabulary.PropertyPreview, vocabulary.PropertyID)
}

// PreviewObject returns a fully object for the "Preview" property
func (object Object) PreviewObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyPreview, vocabulary.PropertyID)
}

// Published representsthe date and time at which the object was published
func (object Object) Published() time.Time {
	return object.ParseTime(vocabulary.PropertyPublished)
}

// PublishedObject returns a fully object for the "Published" property
func (object Object) PublishedObject() time.Time {
	return object.ParseTime(vocabulary.PropertyPublished)
}

// Replies identifies a Collection containing objects considered to be responses to this object.
func (object Object) Replies() Collection {
	return Collection{}
}

// StartTime represents the date and time describing the actual or expected starting time of the object. When used with an Activity object, for instance, the startTime property specifies the moment the activity began or is scheduled to begin
func (object Object) StartTime() time.Time {
	return object.ParseTime(vocabulary.PropertyStartTime)
}

// StartTimeObject returns a fully object for the "StartTime" property
func (object Object) StartTimeObject() time.Time {
	return object.ParseTime(vocabulary.PropertyStartTime)
}

// Summary is a natural language summarization of the object encoded as HTML. Multiple language tagged summaries MAY be provided.
func (object Object) Summary(language ...string) string {
	return object.ParseMap(vocabulary.PropertySummary, language...)
}

// Tag represents one or more "tags" that have been associated with an objects. A tag can be any kind of Object. The key difference between attachment and tag is that the former implies association by inclusion, while the latter implies associated by reference.
func (object Object) Tag() string {
	return object.ParseString(vocabulary.PropertyTag, vocabulary.PropertyID)
}

// TagObject returns a fully object for the "Tag" property
func (object Object) TagObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyTag, vocabulary.PropertyID)
}

// Updated represents the date and time at which the object was updated
func (object Object) Updated() time.Time {
	return object.ParseTime(vocabulary.PropertyUpdated)
}

// UpdatedObject returns a fully object for the "Updated" property
func (object Object) UpdatedObject() time.Time {
	return object.ParseTime(vocabulary.PropertyUpdated)
}

// URL identifies one or more links to representations of the object
func (object Object) URL() string {
	return object.ParseString(vocabulary.PropertyURL, vocabulary.PropertyHREF)
}

// URLObject returns a fully object for the "URL" property
func (object Object) URLObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyURL, vocabulary.PropertyHREF)
}

// To identifies an entity considered to be part of the public primary audience of an Object
func (object Object) To() string {
	return object.ParseString(vocabulary.PropertyTo, vocabulary.PropertyID)
}

// ToObject returns a fully object for the "To" property
func (object Object) ToObject() (*Object, bool) {
	return object.ParseObject(vocabulary.PropertyTo, vocabulary.PropertyID)
}

// MediaType identifies the MIME media type of the referenced resource.  When used on an Object, identifies the MIME media type of the value of the content property. If not specified, the content property is assumed to contain text/html content.
func (object Object) MediaType() string {
	return object.ParseString(vocabulary.PropertyMediaType, "")
}

// MediaTypeObject returns a fully object for the "MediaType" property
func (object Object) MediaTypeObject() string {
	return object.ParseString(vocabulary.PropertyMediaType, "")
}

// HrefLang hints as to the language used by the target resource. Value MUST be a [BCP47] Language-Tag.
func (object Object) HrefLang() string {
	return object.ParseString(vocabulary.PropertyHrefLang, "")
}

// Rel represents a link relation associated with a Link. The value MUST conform to both the [HTML5] and [RFC5988] "link relation" definitions.
func (object Object) Rel() string {
	return object.ParseString(vocabulary.PropertyRel, "")
}

// Height specifies a hint as to the rendering height in device-independent pixels of the linked resource.
func (object Object) Height() (int64, bool) {
	return object.ParseInt(vocabulary.PropertyHeight)
}

// Width specifies a hint as to the rendering width in device-independent pixels of the linked resource.
func (object Object) Width() (int64, bool) {
	return object.ParseInt(vocabulary.PropertyWidth)
}

/*** Parser Functions ***/

// ParseContext tries to identify a @context property within the data structure.
func (object Object) ParseContext() (Context, bool) {
	return Context{}, false
}

// ParseObject tries to identify an embedded ActivityStream object within a specific object property
func (object Object) ParseObject(property string, defaultProperty string) (*Object, bool) {

	if value, ok := object.currentRecord[property]; ok {

		if first, rest, ok := parseObject(value, defaultProperty); ok {

			object.currentRecord = first
			object.nextRecords = rest
			return &object, true
		}
	}

	// Fall through means that we don't have a valid property.
	return nil, false
}

func parseObject(value interface{}, defaultProperty string) (first map[string]interface{}, rest []interface{}, ok bool) {

	// Initialize named result values
	first = map[string]interface{}{}
	rest = []interface{}{}

	switch value := value.(type) {

	case string:

		if defaultProperty != "" {
			first[defaultProperty] = value
			return first, rest, true
		}

	case map[string]interface{}:
		first = value
		return first, rest, true

	case []interface{}:

		if len(value) > 0 {

			if f, _, ok := parseObject(value[0], defaultProperty); ok {
				first = f
			}

			if len(value) > 1 {
				rest = value[1:]
			}

			return first, rest, true
		}
	}

	// Fall through to here means that we could not parse the value
	return first, rest, false
}

// ParseString tries to find the string value of a property
func (object Object) ParseString(property string, defaultProperty string) string {

	if value, ok := object.currentRecord[property]; ok {
		return parseString(value, defaultProperty)
	}

	// Fall through to here means that we could not find an appropriate value for this property.
	return ""
}

// ParseString tries to find the string value of a property
func parseString(value interface{}, defaultProperty string) string {

	switch value := value.(type) {

	case string:
		return value

	case map[string]interface{}:

		if defaultProperty != "" {

			// If MAP, then look up the default property value within the map.
			if value, ok := value[defaultProperty]; ok {
				return parseString(value, "")
			}
		}

	case []interface{}:

		// If ARRAY, then check that it is not empty
		if len(value) > 0 {
			return parseString(value[0], defaultProperty)
		}
	}

	// Fall through to here means that we could not find an appropriate value for this property.
	return ""
}

// ParseStringArray tries to find a property with a string array value.
func (object Object) ParseStringArray(property string) []string {

	if value, ok := object.currentRecord[property]; ok {

		switch value := value.(type) {

		case string:
			return []string{value}

		case []interface{}:

			if len(value) > 0 {

				result := []string{}

				for _, item := range value {

					if item, ok := item.(string); ok {
						result = append(result, item)
					}
				}

				return result
			}
		}
	}

	return []string{}
}

// ParseInt searches the object for a matching property, and parses this value as an Integer
func (object Object) ParseInt(property string) (int64, bool) {

	if value, ok := object.currentRecord[property]; ok {

		switch value := value.(type) {

		case int:
			return int64(value), true

		case int16:
			return int64(value), true

		case int32:
			return int64(value), true

		case int64:
			return value, true

		case float32:
			return int64(value), true

		case float64:
			return int64(value), true

		case string:
			if result, err := strconv.ParseInt(value, 10, 64); err != nil {
				return result, true
			}
		}
	}

	return int64(0), false
}

// ParseMap searches the object for a propertyMap value that matches one of the provided languages.  If no language is provided, then
// the default language is used.
func (object Object) ParseMap(property string, languages ...string) string {

	defaultLanguage := "en"
	propertyMap := property + "Map"

	// If a multi-language "Map" exists, then see if we can do better than the default
	if valueMap, ok := object.currentRecord[propertyMap]; ok {

		// Guarantee that valueMap is the correct type
		if valueMap, ok := valueMap.(map[string]interface{}); ok {

			// Add the default language to the end of the language search
			languages = append(languages, defaultLanguage)

			// Loop over all available languages
			for _, language := range languages {

				// If a value exists for this language, then use it.
				if value, ok := valueMap[language]; ok {

					// Guarantee that value is the right type
					if value, ok := value.(string); ok {
						return value
					}
				}
			}
		}
	}

	// If none found, then search for the value in the regular (non-map) property
	if value, ok := object.currentRecord[property]; ok {

		// Guarantee that the value is the correct type
		if value, ok := value.(string); ok {
			return value
		}
	}

	return ""
}

// ParseTime tries to identify a time-based property
func (object Object) ParseTime(property string) time.Time {
	return time.Time{}
}

// ParseDuration tries to identify a duration-based property
func (object Object) ParseDuration(property string) time.Duration {
	return time.Duration(0)
}

// UnmarshalJSON implements a custom JSON unmarshaller.
func (object *Object) UnmarshalJSON(input []byte) error {

	var currentRecord map[string]interface{}

	if err := json.Unmarshal(input, &(object.currentRecord)); err != nil {
		return derp.New(http.StatusBadRequest, "activitystream.object", "Unable to unmarshal JSON", err, input)
	}

	object.currentRecord = currentRecord
	object.nextRecords = []interface{}{}

	return nil
}
