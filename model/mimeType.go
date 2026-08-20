package model

// MimeTypeActivityPub is the content type of an ActivityPub document
const MimeTypeActivityPub = "application/activity+json"

// MimeTypeAtom is the content type of an Atom feed
const MimeTypeAtom = "application/atom+xml"

// MimeTypeEventStream is the content type of a Server-Sent Events stream
const MimeTypeEventStream = "text/event-stream"

// MimeTypeJSON is the content type of a plain JSON document
const MimeTypeJSON = "application/json"

// MimeTypeJSONLD is the content type of a JSON-LD document
const MimeTypeJSONLD = "application/ld+json"

// MimeTypeJSONLDWithProfile is the content type of a JSON-LD document in the ActivityStreams profile
const MimeTypeJSONLDWithProfile = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`

// MimeTypeJSONFeed is the content type of a JSONFeed document
const MimeTypeJSONFeed = "application/feed+json"

// https://datatracker.ietf.org/doc/html/rfc7033#section-10.2
const MimeTypeJSONResourceDescriptor = "application/jrd+json"

// https://datatracker.ietf.org/doc/html/rfc7033#section-10.2
// With charset extension to match Mastodon
const MimeTypeJSONResourceDescriptorWithCharset = "application/jrd+json; charset=utf-8"

// MimeTypeHTML is the content type of an HTML document
const MimeTypeHTML = "text/html"

// MimeTypeImage matches any image content type
const MimeTypeImage = "image/*"

// MimeTypeRSS is the content type of an RSS feed
const MimeTypeRSS = "application/rss+xml"

// MimeTypeXML is the content type of an XML document
const MimeTypeXML = "application/xml"

// MimeTypeXMLText is the legacy content type of an XML document
const MimeTypeXMLText = "text/xml"
