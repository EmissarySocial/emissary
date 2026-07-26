package model

import (
	"net/url"
	"strings"
)

// HashtagURLPrefix returns the link prefix for #hashtags, built from the "tagUrl"
// defined by a Template (denormalized onto Users as `User.TagURL`).
//
// Hashtag URLs are ABSOLUTE: federated documents are read on other servers, where a
// relative URL cannot be resolved.  Templates still store `tagUrl` as a path prefix
// ("/search?q="), because one server hosts many domains from one set of Templates,
// so the hostname is added here instead.  A prefix that already names a host is used
// as-is, and an empty tagURL returns an empty string, meaning "do not link."
func HashtagURLPrefix(hostname string, tagURL string) string {

	// RULE: An empty TagURL means "extract, but do not linkify"
	if tagURL == "" {
		return ""
	}

	// RULE: Relative TagURLs are anchored to the hostname
	if strings.HasPrefix(tagURL, "/") {
		return hostname + tagURL
	}

	return tagURL
}

// HashtagURL returns the complete link target for a single #hashtag, or an empty
// string when the Template defines no tagURL.  The tag is escaped exactly as
// `replace.Linkify` escapes it, so links published in a document's metadata match
// the anchors written into the document's content.
func HashtagURL(hostname string, tagURL string, tag string) string {

	prefix := HashtagURLPrefix(hostname, tagURL)

	if prefix == "" {
		return ""
	}

	return prefix + "%23" + url.QueryEscape(tag)
}
