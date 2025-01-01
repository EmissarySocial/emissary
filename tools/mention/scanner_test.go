package mention

import (
	"testing"

	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

func TestMention_Beginning(t *testing.T) {

	testMessage := "#This is a tag at the beginning of the message"

	{
		tags, remainder := New().Parse(testMessage)
		require.Equal(t, sliceof.String{"this"}, tags)
		require.Equal(t, " is a tag at the beginning of the message", remainder)
	}

	{
		tags := New().ParseTagsOnly(testMessage)
		require.Equal(t, sliceof.String{"this"}, tags)
	}
}

func TestMention_Ending(t *testing.T) {

	testMessage := "This is a tag at the ending of the @message"

	{
		tags, remainder := New(Mentions()).Parse(testMessage)
		require.Equal(t, sliceof.String{"message"}, tags)
		require.Equal(t, "This is a tag at the ending of the ", remainder)
	}

	{
		tags := New(Mentions()).ParseTagsOnly(testMessage)
		require.Equal(t, sliceof.String{"message"}, tags)
	}
}

func TestHashtags(t *testing.T) {

	testMessage := "#This is a tag at the beginning of the message"

	{
		tags, remainder := New(Hashtags()).Parse(testMessage)
		require.Equal(t, sliceof.String{"this"}, tags)
		require.Equal(t, " is a tag at the beginning of the message", remainder)
	}

	{
		tags := New(Hashtags()).ParseTagsOnly(testMessage)
		require.Equal(t, sliceof.String{"this"}, tags)
	}
}

func TestLong(t *testing.T) {

	testMessage := "This is a #story of a #lovely_lady who was living with #three #very #lovely_girls.\n"
	testMessage += "They all had hair of #gold, like their @mother. The #youngest one in #curls"

	{
		tags, remainder := New(Mentions()).Parse(testMessage)
		require.Equal(t, sliceof.String{"mother"}, tags)
		require.Equal(t, "This is a #story of a #lovely_lady who was living with #three #very #lovely_girls.\nThey all had hair of #gold, like their . The #youngest one in #curls", remainder)
	}

	{
		tags, remainder := New(Hashtags()).Parse(testMessage)
		require.Equal(t, sliceof.String{"story", "lovely_lady", "three", "very", "lovely_girls", "gold", "youngest", "curls"}, tags)
		require.Equal(t, "This is a  of a  who was living with   .\nThey all had hair of , like their @mother. The  one in ", remainder)
	}
}

func TestCombined(t *testing.T) {

	testMessage := "This is a #story of a @lovely_lady who was living with #three #very #lovely_girls.\n"
	testMessage += "They all had hair of #gold, like their @mother. The @youngest_one in #curls"

	{
		tags, _ := New(WithPrefixes('@', '#'), IncludePrefix()).Parse(testMessage)
		require.Equal(t, sliceof.String{"#story", "@lovely_lady", "#three", "#very", "#lovely_girls", "#gold", "@mother", "@youngest_one", "#curls"}, tags)
	}
	{
		tags := New(WithPrefixes('@', '#'), IncludePrefix()).ParseTagsOnly(testMessage)
		require.Equal(t, sliceof.String{"#story", "@lovely_lady", "#three", "#very", "#lovely_girls", "#gold", "@mother", "@youngest_one", "#curls"}, tags)
	}
}

func TestCaseSensitive(t *testing.T) {
	testMessage := "#LoL #YOLO #tags #bRo"

	{
		tags, _ := New(Hashtags()).Parse(testMessage)
		require.Equal(t, sliceof.String{"lol", "yolo", "tags", "bro"}, tags)
	}
	{
		tags, _ := New(Hashtags(), CaseSensitive()).Parse(testMessage)
		require.Equal(t, sliceof.String{"LoL", "YOLO", "tags", "bRo"}, tags)
	}
}

func TestIncludePrefix(t *testing.T) {
	testMessage := "#LoL #YOLO #tags #bRo"

	{
		tags, _ := New(Hashtags(), IncludePrefix()).Parse(testMessage)
		require.Equal(t, sliceof.String{"#lol", "#yolo", "#tags", "#bro"}, tags)
	}

	{
		tags := New(Hashtags(), IncludePrefix()).ParseTagsOnly(testMessage)
		require.Equal(t, sliceof.String{"#lol", "#yolo", "#tags", "#bro"}, tags)
	}
}

func TestIncludePrefix_Mentions(t *testing.T) {
	testMessage := "@john. @sarah, @kyle: @jane; ignore this remainder text."

	{
		tags, _ := New(Mentions(), IncludePrefix()).Parse(testMessage)
		require.Equal(t, sliceof.String{"@john", "@sarah", "@kyle", "@jane"}, tags)
	}
}

func TestWithTerminators(t *testing.T) {

	testMessage := "#Standard #ThisTag.Doesnt,Have:Default;Terminators? "
	{
		tags, _ := New(Hashtags(), CaseSensitive(), WithTerminators(' ')).Parse(testMessage)
		require.Equal(t, sliceof.String{"Standard", "ThisTag.Doesnt,Have:Default;Terminators?"}, tags)
	}

	{
		tags := New(Hashtags(), CaseSensitive(), WithTerminators(' ')).ParseTagsOnly(testMessage)
		require.Equal(t, sliceof.String{"Standard", "ThisTag.Doesnt,Have:Default;Terminators?"}, tags)
	}
}

func TestInTheMiddleOfATag(t *testing.T) {

	testMessage := "This is a #tag#tag with a @username@server.social in the middle of it."
	{
		tags := ParseTagsOnly('#', testMessage)
		require.Equal(t, sliceof.String{"tag#tag"}, tags)
	}

	{
		tags := ParseTagsOnly('@', testMessage)
		require.Equal(t, sliceof.String{"username@server.social"}, tags)
	}
}

func TestNewlines(t *testing.T) {

	{
		testMessage := "This is a tag after a\n#newline"
		tags := ParseTagsOnly('#', testMessage)
		require.Equal(t, sliceof.String{"newline"}, tags)
	}

	{
		testMessage := "This is a tag #before\n a newline"
		tags := ParseTagsOnly('#', testMessage)
		require.Equal(t, sliceof.String{"before"}, tags)
	}
}
