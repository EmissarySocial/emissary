package model

import (
	"github.com/benpate/rosetta/sliceof"
)

// TagList is a document's complete list of Tags, of mixed types (#hashtags, @mentions).
//
// This is an ALIAS rather than a defined type, deliberately.  A defined type would not inherit
// sliceof.Object's methods, and the schema package requires several of them -- Length, GetIndex,
// SetIndex, GetPointer, SetValue -- for any value used as a schema.Array property.  Re-declaring
// them here bought nothing except the chance to drift from rosetta.  As an alias, TagList gets all
// of them (plus Filter, Find, Range, IsEmpty, Append, and the rest) for free.
//
// The cost is that the helpers below are package functions rather than methods, because Go cannot
// define methods on an aliased type from another package.
type TagList = sliceof.Object[Tag]

// NewTagList returns a fully initialized (empty) TagList.
func NewTagList() TagList {
	return make(TagList, 0)
}

// TagsOfType returns only the Tags matching the provided AS2 link type, in their original order.
func TagsOfType(tags TagList, tagType string) TagList {

	result := make(TagList, 0, len(tags))

	for _, tag := range tags {
		if tag.Type == tagType {
			result = append(result, tag)
		}
	}

	return result
}

// TagNames returns the bare names of every Tag of the provided type.  This is the form that search
// indexing, rule matching, and content linkification consume.
func TagNames(tags TagList, tagType string) sliceof.String {

	result := make(sliceof.String, 0, len(tags))

	for _, tag := range tags {
		if tag.Type == tagType {
			result = append(result, tag.Name)
		}
	}

	return result
}

// ReplaceTagsOfType returns a copy of the TagList with every Tag of the provided type removed and
// the replacements appended.  Tags of other types keep their relative order, so one kind of tag can
// be recalculated without disturbing the others.
func ReplaceTagsOfType(tags TagList, tagType string, replacements TagList) TagList {

	result := make(TagList, 0, len(tags)+len(replacements))

	for _, tag := range tags {
		if tag.Type != tagType {
			result = append(result, tag)
		}
	}

	return append(result, replacements...)
}
