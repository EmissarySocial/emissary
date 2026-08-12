package parse

import (
	"testing"

	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// TestHashtag_Beginning confirms that a tag is found at the very start of a message
func TestHashtag_Beginning(t *testing.T) {

	testMessage := "#This is a tag at the beginning of the message"

	tokens, remainder := HashtagsAndRemainder(testMessage)
	require.Equal(t, sliceof.String{"This"}, tokens)
	require.Equal(t, " is a tag at the beginning of the message", remainder)
}

// TestMention_Ending confirms that a tag is found at the very end of a message
func TestMention_Ending(t *testing.T) {

	testMessage := "This is a tag at the ending of the @message"

	tokens, remainder := MentionsAndRemainder(testMessage)
	require.Equal(t, sliceof.String{"message"}, tokens)
	require.Equal(t, "This is a tag at the ending of the ", remainder)
}

// TestHashtag_SkipMiddle confirms that a prefix in the middle of a word does not start a tag
func TestHashtag_SkipMiddle(t *testing.T) {
	testMessage := "This has no#hashtags in it."
	tokens, remainder := HashtagsAndRemainder(testMessage)
	require.Zero(t, len(tokens))
	require.Equal(t, testMessage, remainder)
}

// TestMentions_Long scans a multi-line message for @mentions, ignoring its #hashtags
func TestMentions_Long(t *testing.T) {

	testMessage := "This is a #story of a #lovely_lady who was living with #three #very #lovely_girls.\n"
	testMessage += "They all had hair of #gold, like their @mother. The #youngest one in #curls"

	tokens, remainder := MentionsAndRemainder(testMessage)
	require.Equal(t, sliceof.String{"mother"}, tokens)
	require.Equal(t, "This is a #story of a #lovely_lady who was living with #three #very #lovely_girls.\nThey all had hair of #gold, like their . The #youngest one in #curls", remainder)
}

// TestHashtags_Long scans a multi-line message for #hashtags, ignoring its @mentions
func TestHashtags_Long(t *testing.T) {

	testMessage := "This is a #story of a #lovely_lady who was living with #three #very #lovely_girls.\n"
	testMessage += "They all had hair of #gold, like their @mother. The #youngest one in #curls"

	tokens, remainder := HashtagsAndRemainder(testMessage)
	require.Equal(t, sliceof.String{"story", "lovely_lady", "three", "very", "lovely_girls", "gold", "youngest", "curls"}, tokens)
	require.Equal(t, "This is a  of a  who was living with   .\nThey all had hair of , like their @mother. The  one in ", remainder)
}

// TestCombined scans a message for both prefixes at once, keeping each prefix in the result
func TestCombined(t *testing.T) {

	testMessage := "This is a #story of a @lovely_lady who was living with #three #very #lovely_girls.\n"
	testMessage += "They all had hair of #gold, like their @mother. The @youngest_one in #curls"

	tokens, remainder := All(testMessage, WithIncludePrefix())
	require.Equal(t, sliceof.String{"#story", "@lovely_lady", "#three", "#very", "#lovely_girls", "#gold", "@mother", "@youngest_one", "#curls"}, tokens)
	require.Equal(t, "This is a  of a  who was living with   .\nThey all had hair of , like their . The  in ", remainder)
}

// TestRegression confirms that comma-separated tags are each scanned separately
func TestRegression(t *testing.T) {
	testMessage := "#all, #rock, #funky, #chicken"
	tokens := Hashtags(testMessage)
	require.Equal(t, sliceof.String{"all", "rock", "funky", "chicken"}, tokens)
}

// Hashtags may contain digits, and a leading digit does not split the token.
func TestHashtags_WithDigits(t *testing.T) {
	testMessage := "Testing hashtags #travel #Food2024 #2024recap here"
	tokens := Hashtags(testMessage)
	require.Equal(t, sliceof.String{"travel", "Food2024", "2024recap"}, tokens)
}

// Demonstrates that the "default" setting is Case Sensitivity
func TestCaseSensitive_Default(t *testing.T) {
	testMessage := "#LoL #YOLO #tokens #bRo"

	tokens := New(WithHashtagsOnly()).Parse(testMessage)
	require.Equal(t, sliceof.String{"LoL", "YOLO", "tokens", "bRo"}, tokens)
}

// TestCaseSensitive confirms that tags keep their original case when the option is set
func TestCaseSensitive(t *testing.T) {
	testMessage := "#LoL #YOLO #tokens #bRo"

	tokens := New(WithHashtagsOnly(), WithCaseSensitive()).Parse(testMessage)
	require.Equal(t, sliceof.String{"LoL", "YOLO", "tokens", "bRo"}, tokens)
}

// TestCaseInSensitive confirms that tags are lower-cased when the option is set
func TestCaseInSensitive(t *testing.T) {
	testMessage := "#LoL #YOLO #tokens #bRo"

	tokens := New(WithHashtagsOnly(), WithCaseInsensitive()).Parse(testMessage)
	require.Equal(t, sliceof.String{"lol", "yolo", "tokens", "bro"}, tokens)
}

// TestIncludePrefix confirms that '#' is kept in the result when the option is set
func TestIncludePrefix(t *testing.T) {
	testMessage := "#lol #yolo #tokens #bro"

	tokens := New(WithHashtagsOnly(), WithIncludePrefix()).Parse(testMessage)
	require.Equal(t, sliceof.String{"#lol", "#yolo", "#tokens", "#bro"}, tokens)
}

// TestIncludePrefix_Mentions confirms that '@' is kept in the result when the option is set
func TestIncludePrefix_Mentions(t *testing.T) {
	testMessage := "@john. @sarah, @kyle: @jane; ignore this remainder text. #ignore-hashtag"

	tokens := New(WithMentionsOnly(), WithIncludePrefix()).Parse(testMessage)
	require.Equal(t, sliceof.String{"@john", "@sarah", "@kyle", "@jane"}, tokens)
}

// TestWeirdTerminators confirms which punctuation ends a hashtag mid-word, and which does not
func TestWeirdTerminators(t *testing.T) {

	testMessage := "#Standard #ThisTag.Sure,Has:Weird;Terminators? "

	tokens := New(WithHashtagsOnly(), WithCaseSensitive()).Parse(testMessage)
	require.Equal(t, sliceof.String{"Standard", "ThisTag.Sure"}, tokens)
}

// TestSoftTerminators1 confirms that a period ends a hashtag only at a word boundary
func TestSoftTerminators1(t *testing.T) {

	testMessage := "#One #Two. #Three.Four"

	tokens := Hashtags(testMessage)
	require.Equal(t, sliceof.String{"One", "Two", "Three.Four"}, tokens)
}

// TestSoftTerminators2 confirms that a trailing period does not truncate a WebFinger handle
func TestSoftTerminators2(t *testing.T) {

	testMessage := "Please tell @username@server.social. It's important."

	tokens := Mentions(testMessage)
	require.Equal(t, sliceof.String{"username@server.social"}, tokens)
}

// TestSoftTerminators3 confirms that commas and question marks end a mention outright
func TestSoftTerminators3(t *testing.T) {

	testMessage := "Please tell @username@server.social, it's important, okay @username2?"

	tokens := Mentions(testMessage)
	require.Equal(t, sliceof.String{"username@server.social", "username2"}, tokens)
}

// TestInTheMiddleOfATag pins the asymmetry between the two prefixes: '#' cannot appear inside a
// hashtag, so "#tag#tag" is two tags -- but '@' CAN appear inside a mention, so
// "@username@server.social" stays one.  This test previously asserted "tag#tag", which was the
// defect in BUG-69 rather than the intended behavior.
func TestInTheMiddleOfATag(t *testing.T) {

	testMessage := "This is a #tag#tag with a @username@server.social in the middle of it."

	tags := Hashtags(testMessage)
	require.Equal(t, sliceof.String{"tag", "tag"}, tags)

	mentions := Mentions(testMessage)
	require.Equal(t, sliceof.String{"username@server.social"}, mentions)
}

// TestNewlines confirms that a tag may begin immediately after a line break
func TestNewlines(t *testing.T) {

	testMessage := "This is a tag after a\n#newline"
	tokens := Hashtags(testMessage)
	require.Equal(t, sliceof.String{"newline"}, tokens)
}

/******************************************
 * BUG-69 — Hashtag Terminators
 ******************************************/

// TestBug69_HashtagTerminators covers the punctuation that every other fediverse server breaks a
// tag on.  The hyphen is the one that matters in practice: "#well-being" is "#well" everywhere else,
// so a tag published with the hyphen inside it is a tag nobody else indexes.
func TestBug69_HashtagTerminators(t *testing.T) {

	table := []struct {
		input    string
		expected sliceof.String
	}{
		{"#well-being", sliceof.String{"well"}},
		{"#foo#bar", sliceof.String{"foo", "bar"}},
		{"#tag@user", sliceof.String{"tag"}},
		{"#a&b", sliceof.String{"a"}},
		{"#a^b", sliceof.String{"a"}},
		{"#a~b", sliceof.String{"a"}},
		{"#a=b", sliceof.String{"a"}},
		{"#a*b", sliceof.String{"a"}},
		{"#a$b", sliceof.String{"a"}},
	}

	for _, test := range table {
		require.Equal(t, test.expected, Hashtags(test.input), test.input)
	}
}

// TestBug69_HashtagRegressions guards the characters that must NOT terminate a tag.  Switching to
// FEP-eb48 rule 2's literal allow-list ("A-Z, a-z, 0-9") would break every one of the non-ASCII
// cases here, which is why the parser stays a deny-list.
func TestBug69_HashtagRegressions(t *testing.T) {

	require.Equal(t, sliceof.String{"snake_case"}, Hashtags("#snake_case"))
	require.Equal(t, sliceof.String{"日本語"}, Hashtags("#日本語"))
	require.Equal(t, sliceof.String{"café"}, Hashtags("#café"))
	require.Equal(t, sliceof.String{"Привет"}, Hashtags("#Привет"))
	require.Equal(t, sliceof.String{"tag2024"}, Hashtags("#tag2024"))

	// A soft terminator still ends the tag only at a word boundary
	require.Equal(t, sliceof.String{"hashtag"}, Hashtags("#hashtag."))
	require.Equal(t, sliceof.String{"a.b"}, Hashtags("#a.b"))
}

// TestBug69_PrecedingPunctuation covers FEP-eb48's preceding-punctuation examples. These found
// NOTHING before this fix -- a tag could only start after whitespace, so any '#' preceded by
// punctuation was skipped entirely, including the very common "(#hashtag)".
func TestBug69_PrecedingPunctuation(t *testing.T) {

	for _, prefix := range []string{"-", "&", "^", "~", "=", "*", "$", "(", "[", "{", "<", ",", "!", "?", ";", "%", "\"", "'"} {
		require.Equal(t, sliceof.String{"hashtag"}, Hashtags(prefix+"#hashtag"), prefix)
	}

	require.Equal(t, sliceof.String{"hashtag"}, Hashtags("(#hashtag)"))

	// A tag still may NOT start in the middle of a word
	require.Equal(t, sliceof.String{}, Hashtags("a#hashtag"))
}

// TestBug69_MentionsUnaffected is the guard that makes the per-prefix terminator sets necessary.
// '@' and '-' terminate a hashtag but must never terminate a mention, because a WebFinger handle
// needs both.  A single shared list cannot satisfy both, which is why the lists moved onto Parser.
func TestBug69_MentionsUnaffected(t *testing.T) {

	require.Equal(t, sliceof.String{"user@host.social"}, Mentions("@user@host.social"))
	require.Equal(t, sliceof.String{"user@my-host.social"}, Mentions("@user@my-host.social"))
	require.Equal(t, sliceof.String{"first-last@sub.domain.example"}, Mentions("@first-last@sub.domain.example"))
	require.Equal(t, sliceof.String{"bob"}, Mentions("@bob"))

	// An email address is still not a mention: a token may not begin mid-word
	require.Equal(t, sliceof.String{}, Mentions("bob@example.com"))
}

// TestBug69_PeekNextRuneMultibyte covers the soft-terminator lookahead across a multi-byte rune.
// peekNextRune byte-indexed and cast to rune before this fix, so the offset landed mid-rune
// whenever the terminator followed non-ASCII text.
func TestBug69_PeekNextRuneMultibyte(t *testing.T) {

	require.Equal(t, sliceof.String{"日本語"}, Hashtags("#日本語. "))
	require.Equal(t, sliceof.String{"café"}, Hashtags("#café. "))
	require.Equal(t, sliceof.String{"café.beans"}, Hashtags("#café.beans"))
	require.Equal(t, sliceof.String{"user@café.social"}, Mentions("@user@café.social"))
}
