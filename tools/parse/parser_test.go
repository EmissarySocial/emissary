package parse

import (
	"testing"

	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

func TestHashtag_Beginning(t *testing.T) {

	testMessage := "#This is a tag at the beginning of the message"

	tokens, remainder := HashtagsAndRemainder(testMessage)
	require.Equal(t, sliceof.String{"This"}, tokens)
	require.Equal(t, " is a tag at the beginning of the message", remainder)
}

func TestMention_Ending(t *testing.T) {

	testMessage := "This is a tag at the ending of the @message"

	tokens, remainder := MentionsAndRemainder(testMessage)
	require.Equal(t, sliceof.String{"message"}, tokens)
	require.Equal(t, "This is a tag at the ending of the ", remainder)
}

func TestHashtag_SkipMiddle(t *testing.T) {
	testMessage := "This has no#hashtags in it."
	tokens, remainder := HashtagsAndRemainder(testMessage)
	require.Zero(t, len(tokens))
	require.Equal(t, testMessage, remainder)
}

func TestMentions_Long(t *testing.T) {

	testMessage := "This is a #story of a #lovely_lady who was living with #three #very #lovely_girls.\n"
	testMessage += "They all had hair of #gold, like their @mother. The #youngest one in #curls"

	tokens, remainder := MentionsAndRemainder(testMessage)
	require.Equal(t, sliceof.String{"mother"}, tokens)
	require.Equal(t, "This is a #story of a #lovely_lady who was living with #three #very #lovely_girls.\nThey all had hair of #gold, like their . The #youngest one in #curls", remainder)
}

func TestHashtags_Long(t *testing.T) {

	testMessage := "This is a #story of a #lovely_lady who was living with #three #very #lovely_girls.\n"
	testMessage += "They all had hair of #gold, like their @mother. The #youngest one in #curls"

	tokens, remainder := HashtagsAndRemainder(testMessage)
	require.Equal(t, sliceof.String{"story", "lovely_lady", "three", "very", "lovely_girls", "gold", "youngest", "curls"}, tokens)
	require.Equal(t, "This is a  of a  who was living with   .\nThey all had hair of , like their @mother. The  one in ", remainder)
}

func TestCombined(t *testing.T) {

	testMessage := "This is a #story of a @lovely_lady who was living with #three #very #lovely_girls.\n"
	testMessage += "They all had hair of #gold, like their @mother. The @youngest_one in #curls"

	tokens, remainder := All(testMessage, WithIncludePrefix())
	require.Equal(t, sliceof.String{"#story", "@lovely_lady", "#three", "#very", "#lovely_girls", "#gold", "@mother", "@youngest_one", "#curls"}, tokens)
	require.Equal(t, "This is a  of a  who was living with   .\nThey all had hair of , like their . The  in ", remainder)
}

func TestRegression(t *testing.T) {
	testMessage := "#all, #rock, #funky, #chicken"
	tokens := Hashtags(testMessage)
	require.Equal(t, sliceof.String{"all", "rock", "funky", "chicken"}, tokens)
}

// Demonstrates that the "default" setting is Case Sensitivity
func TestCaseSensitive_Default(t *testing.T) {
	testMessage := "#LoL #YOLO #tokens #bRo"

	tokens := New(WithHashtagsOnly()).Parse(testMessage)
	require.Equal(t, sliceof.String{"LoL", "YOLO", "tokens", "bRo"}, tokens)
}

func TestCaseSensitive(t *testing.T) {
	testMessage := "#LoL #YOLO #tokens #bRo"

	tokens := New(WithHashtagsOnly(), WithCaseSensitive()).Parse(testMessage)
	require.Equal(t, sliceof.String{"LoL", "YOLO", "tokens", "bRo"}, tokens)
}

func TestCaseInSensitive(t *testing.T) {
	testMessage := "#LoL #YOLO #tokens #bRo"

	tokens := New(WithHashtagsOnly(), WithCaseInsensitive()).Parse(testMessage)
	require.Equal(t, sliceof.String{"lol", "yolo", "tokens", "bro"}, tokens)
}

func TestIncludePrefix(t *testing.T) {
	testMessage := "#lol #yolo #tokens #bro"

	tokens := New(WithHashtagsOnly(), WithIncludePrefix()).Parse(testMessage)
	require.Equal(t, sliceof.String{"#lol", "#yolo", "#tokens", "#bro"}, tokens)
}

func TestIncludePrefix_Mentions(t *testing.T) {
	testMessage := "@john. @sarah, @kyle: @jane; ignore this remainder text. #ignore-hashtag"

	tokens := New(WithMentionsOnly(), WithIncludePrefix()).Parse(testMessage)
	require.Equal(t, sliceof.String{"@john", "@sarah", "@kyle", "@jane"}, tokens)
}

func TestWeirdTerminators(t *testing.T) {

	testMessage := "#Standard #ThisTag.Sure,Has:Weird;Terminators? "

	tokens := New(WithHashtagsOnly(), WithCaseSensitive()).Parse(testMessage)
	require.Equal(t, sliceof.String{"Standard", "ThisTag.Sure"}, tokens)
}

func TestSoftTerminators1(t *testing.T) {

	testMessage := "#One #Two. #Three.Four"

	tokens := Hashtags(testMessage)
	require.Equal(t, sliceof.String{"One", "Two", "Three.Four"}, tokens)
}

func TestSoftTerminators2(t *testing.T) {

	testMessage := "Please tell @username@server.social. It's important."

	tokens := Mentions(testMessage)
	require.Equal(t, sliceof.String{"username@server.social"}, tokens)
}

func TestSoftTerminators3(t *testing.T) {

	testMessage := "Please tell @username@server.social, it's important, okay @username2?"

	tokens := Mentions(testMessage)
	require.Equal(t, sliceof.String{"username@server.social", "username2"}, tokens)
}

func TestInTheMiddleOfATag(t *testing.T) {

	testMessage := "This is a #tag#tag with a @username@server.social in the middle of it."

	tags := Hashtags(testMessage)
	require.Equal(t, sliceof.String{"tag#tag"}, tags)

	mentions := Mentions(testMessage)
	require.Equal(t, sliceof.String{"username@server.social"}, mentions)
}

func TestNewlines(t *testing.T) {

	testMessage := "This is a tag after a\n#newline"
	tokens := Hashtags(testMessage)
	require.Equal(t, sliceof.String{"newline"}, tokens)
}
