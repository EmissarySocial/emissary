package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

// TagHrefUnresolvable marks a Tag whose Href was looked up and could not be found.
const TagHrefUnresolvable = "-"

// Tag is one entry in a document's ActivityStreams `tag` array -- a #hashtag, an @mention,
// or (in future) a custom emoji.  See projects/TAGS-UNIFICATION.md.
type Tag struct {
	// Type is an AS2 link type: vocab.LinkTypeHashtag or vocab.LinkTypeMention.
	Type string `bson:"type"`

	// Name is the BARE token, with no "#" or "@" prefix -- "Food2024", "bob@server.social".
	// Roughly ten call sites (search indexing, rule matching, linkification) want the bare
	// value; only the emission sites want the prefix, and they use DisplayName().
	Name string `bson:"name"`

	// Href is stored ONLY for tag types whose link target cannot be derived locally.
	//
	// A #hashtag's target IS derivable -- model.HashtagURL() builds it from the current
	// hostname and the Template's TagURL -- so storing it would freeze a live computation
	// into stale data the next time either one changes.  Hashtag entries leave this empty
	// and the link is computed at emission.
	//
	// An @mention's target is NOT derivable: turning "bob@server.social" into an Actor URL
	// requires asking that server.  Mention entries carry the resolved value, or
	// TagHrefUnresolvable once a lookup has failed.
	Href string `bson:"href,omitempty"`
}

// NewTag returns a fully initialized Tag of the given type.
func NewTag(tagType string, name string) Tag {
	return Tag{
		Type: tagType,
		Name: name,
	}
}

// Prefix returns the microsyntax character that introduces this kind of tag in content.
func (tag Tag) Prefix() string {

	switch tag.Type {

	case vocab.LinkTypeHashtag:
		return "#"

	case vocab.LinkTypeMention:
		return "@"
	}

	return ""
}

// DisplayName returns the prefixed form of this Tag -- "#Food2024", "@bob@server.social".
func (tag Tag) DisplayName() string {
	return tag.Prefix() + tag.Name
}

// NeedsResolution returns TRUE if this Tag requires a network lookup that has not happened yet.
func (tag Tag) NeedsResolution() bool {
	return (tag.Type == vocab.LinkTypeMention) && (tag.Href == "")
}

// IsResolved returns TRUE if this Tag carries a usable Href.
func (tag Tag) IsResolved() bool {
	return (tag.Href != "") && (tag.Href != TagHrefUnresolvable)
}

// Link returns this Tag's link target, which is DERIVED for some tag types and STORED for others.
func (tag Tag) Link(hostname string, tagURL string) string {

	// A #hashtag's target is built from the current hostname and the Template's tagUrl, so that
	// changing either one takes effect on every existing document immediately.  An empty tagUrl
	// yields an empty link, which means "extract, but do not linkify".
	if tag.Type == vocab.LinkTypeHashtag {
		return HashtagURL(hostname, tagURL, tag.Name)
	}

	// Every other type carries its own target, which is usable only once it has been resolved.
	if tag.IsResolved() {
		return tag.Href
	}

	return ""
}

// JSONLD returns the AS2 representation of this Tag.  The hostname and Template tagUrl are needed
// because some tag types derive their link rather than storing it -- see Link().
func (tag Tag) JSONLD(hostname string, tagURL string) mapof.String {

	result := mapof.String{
		vocab.PropertyType: tag.Type,
		vocab.PropertyName: tag.DisplayName(),
	}

	// Publish a tag with no link target WITHOUT an href, rather than with an empty one.
	if href := tag.Link(hostname, tagURL); href != "" {
		result[vocab.PropertyHref] = href
	}

	return result
}
