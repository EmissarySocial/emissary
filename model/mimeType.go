package model

const MagicMimeTypeWebSub = "MAGIC-MIME-TYPE-WEBSUB"

const MimeTypeActivityPub = "application/activity+json"

const MimeTypeAtom = "application/atom+xml"

const MimeTypeEventStream = "text/event-stream"

const MimeTypeJSON = "application/json"

const MimeTypeJSONLD = "application/ld+json"

const MimeTypeJSONLDWithProfile = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`

const MimeTypeJSONFeed = "application/feed+json"

// https://datatracker.ietf.org/doc/html/rfc7033#section-10.2
const MimeTypeJSONResourceDescriptor = "application/jrd+json"

// https://datatracker.ietf.org/doc/html/rfc7033#section-10.2
// With charset extension to match Mastodon
const MimeTypeJSONResourceDescriptorWithCharset = "application/jrd+json; charset=utf-8"

const MimeTypeHTML = "text/html"

const MimeTypeImage = "image/*"

const MimeTypeRSS = "application/rss+xml"

const MimeTypeXML = "application/xml"

const MimeTypeXMLText = "text/xml"
