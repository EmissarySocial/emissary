package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestTagList_OfTypeAndNames confirms the filters that let ~10 search/index/linkify call sites keep
// working with bare token strings.
func TestTagList_OfTypeAndNames(t *testing.T) {

	tags := TagList{
		NewTag(vocab.LinkTypeHashtag, "travel"),
		{Type: vocab.LinkTypeMention, Name: "bob@server.social", Href: "https://server.social/@bob"},
		NewTag(vocab.LinkTypeHashtag, "Food2024"),
	}

	require.Equal(t, 2, len(TagsOfType(tags, vocab.LinkTypeHashtag)))
	require.Equal(t, 1, len(TagsOfType(tags, vocab.LinkTypeMention)))
	require.Equal(t, []string{"travel", "Food2024"}, []string(TagNames(tags, vocab.LinkTypeHashtag)), "order is preserved")
	require.Equal(t, []string{"bob@server.social"}, []string(TagNames(tags, vocab.LinkTypeMention)))
	require.Empty(t, TagNames(tags, "Emoji"))
}

// TestTagList_ReplaceType is the primitive that lets hashtags and mentions be recalculated
// independently. Recalculating one kind must not disturb the other.
func TestTagList_ReplaceType(t *testing.T) {

	mention := Tag{Type: vocab.LinkTypeMention, Name: "bob@server.social", Href: "https://server.social/@bob"}

	tags := TagList{
		NewTag(vocab.LinkTypeHashtag, "travel"),
		mention,
		NewTag(vocab.LinkTypeHashtag, "Food2024"),
	}

	result := ReplaceTagsOfType(tags, vocab.LinkTypeHashtag, TagList{NewTag(vocab.LinkTypeHashtag, "coffee")})

	require.Equal(t, TagList{mention, NewTag(vocab.LinkTypeHashtag, "coffee")}, result)
	require.Equal(t, "https://server.social/@bob", result[0].Href, "the surviving mention keeps its resolved href")
}

// TestTagList_ReplaceType_Empty confirms that replacing with nothing removes every tag of that type.
func TestTagList_ReplaceType_Empty(t *testing.T) {

	tags := TagList{
		NewTag(vocab.LinkTypeHashtag, "travel"),
		{Type: vocab.LinkTypeMention, Name: "bob@server.social"},
	}

	result := ReplaceTagsOfType(tags, vocab.LinkTypeHashtag, NewTagList())

	require.Equal(t, 1, len(result))
	require.Equal(t, vocab.LinkTypeMention, result[0].Type)
}

/********************************
 * Schema / Accessor Interfaces
 ********************************/

// TestTagList_ArrayInterfaces exercises the ArrayGetterSetter methods that schema.Array validation
// requires. TagList inherits them from sliceof.Object rather than declaring its own; this test is
// the guard that the alias keeps them reachable.
//
// NOTE: it does NOT probe SetIndex with a negative index. sliceof.Object.SetIndex panics there
// (object.go:217 indexes the slice after a grow loop that no-ops for negative values), and this
// type does not wrap it.
func TestTagList_ArrayInterfaces(t *testing.T) {

	tags := TagList{NewTag(vocab.LinkTypeHashtag, "travel")}

	require.Equal(t, 1, tags.Length())

	value, ok := tags.GetIndex(0)
	require.True(t, ok)
	require.Equal(t, NewTag(vocab.LinkTypeHashtag, "travel"), value)

	_, ok = tags.GetIndex(1)
	require.False(t, ok, "out of range")

	_, ok = tags.GetIndex(-1)
	require.False(t, ok, "negative index")

	// SetIndex grows the list to fit, matching sliceof.Object's behavior
	require.True(t, tags.SetIndex(2, NewTag(vocab.LinkTypeMention, "bob@server.social")))
	require.Equal(t, 3, tags.Length())
	require.Equal(t, "bob@server.social", tags[2].Name)

	require.False(t, tags.SetIndex(0, "not a tag"), "wrong type is refused")
}

// TestTagList_GetPointer confirms the index-based traversal the schema package uses to reach into
// individual Tags, including the "next" and "last" aliases.
func TestTagList_GetPointer(t *testing.T) {

	tags := TagList{NewTag(vocab.LinkTypeHashtag, "travel")}

	pointer, ok := tags.GetPointer("0")
	require.True(t, ok)
	require.Equal(t, "travel", pointer.(*Tag).Name)

	last, ok := tags.GetPointer("last")
	require.True(t, ok)
	require.Equal(t, "travel", last.(*Tag).Name)

	// "next" appends a blank Tag, which is how the schema package builds up an array
	next, ok := tags.GetPointer("next")
	require.True(t, ok)
	require.Equal(t, 2, tags.Length())
	next.(*Tag).Name = "coffee"
	require.Equal(t, "coffee", tags[1].Name)

	_, ok = tags.GetPointer("nope")
	require.False(t, ok)
}

// TestTagList_SetValue confirms whole-list replacement, which schema.Set uses when assigning an
// entire array at once.
func TestTagList_SetValue(t *testing.T) {

	tags := NewTagList()
	replacement := TagList{NewTag(vocab.LinkTypeHashtag, "travel")}

	require.NoError(t, tags.SetValue(replacement))
	require.Equal(t, 1, tags.Length())

	require.NoError(t, tags.SetValue(&replacement))
	require.Equal(t, 1, tags.Length())

	require.NoError(t, tags.SetValue(NewTagList()))
	require.Equal(t, 0, tags.Length())

	require.Error(t, tags.SetValue("nope"))

	// Inherited from sliceof.Object, which accepts only its own type -- a bare []Tag is refused
	// even though it has the same underlying shape. Callers must pass a TagList.
	require.Error(t, tags.SetValue([]Tag{{Type: vocab.LinkTypeHashtag, Name: "travel"}}))
}

// TestTagList_SchemaRoundTrip drives a TagList through the real schema machinery -- the path that
// Stream.Save takes on every write -- reading and writing individual Tag properties by path.
func TestTagList_SchemaRoundTrip(t *testing.T) {

	stream := NewStream()
	stream.TemplateID = "test-post"
	stream.Tags = TagList{
		NewTag(vocab.LinkTypeHashtag, "travel"),
		{Type: vocab.LinkTypeMention, Name: "bob@server.social", Href: "https://server.social/@bob"},
	}

	s := schema.New(StreamSchema())

	name, err := s.Get(&stream, "tags.0.name")
	require.NoError(t, err)
	require.Equal(t, "travel", name)

	href, err := s.Get(&stream, "tags.1.href")
	require.NoError(t, err)
	require.Equal(t, "https://server.social/@bob", href)

	require.NoError(t, s.Set(&stream, "tags.0.name", "coffee"))
	require.Equal(t, "coffee", stream.Tags[0].Name)
}

// TestTagList_SchemaNormalize confirms that every state a Tag can be in survives normalization --
// notably the TagHrefUnresolvable sentinel, which is not a URL and would be rejected if `href`
// carried Format:"url".
func TestTagList_SchemaNormalize(t *testing.T) {

	stream := NewStream()
	stream.TemplateID = "test-post"
	stream.Tags = TagList{
		NewTag(vocab.LinkTypeHashtag, "travel"),
		{Type: vocab.LinkTypeMention, Name: "bob@server.social", Href: "https://server.social/@bob"},
		{Type: vocab.LinkTypeMention, Name: "pending@server.social"},
		{Type: vocab.LinkTypeMention, Name: "nobody@nowhere.invalid", Href: TagHrefUnresolvable},
	}

	_, err := schema.New(StreamSchema()).Normalize(&stream)

	require.NoError(t, err)
	require.Equal(t, 4, stream.Tags.Length())
	require.Equal(t, "", stream.Tags[2].Href, "an unresolved mention survives intact")
	require.Equal(t, TagHrefUnresolvable, stream.Tags[3].Href, "and so does the negative-cache sentinel")
}
