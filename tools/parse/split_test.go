package parse

import (
	"testing"

	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// TestSplit confirms that a string is split into tokens, with any prefix characters removed
func TestSplit(t *testing.T) {

	require.Equal(t, sliceof.String{}, Split(""))
	require.Equal(t, sliceof.String{"hello", "world"}, Split("hello world"))
	require.Equal(t, sliceof.String{"hello", "world"}, Split("hello  world"))
	require.Equal(t, sliceof.String{"hello", "world"}, Split("#hello world"))
	require.Equal(t, sliceof.String{"hello", "world"}, Split("#hello #world"))
	require.Equal(t, sliceof.String{"hello", "world", "people"}, Split("hello, world,people "))
	require.Equal(t, sliceof.String{"hey", "there", "ladies", "and", "gentlemen"}, Split("hey there ladies and gentlemen"))
}

// TestIsEndOfToken confirms which runes end a token for a default, mixed-prefix Parser
func TestIsEndOfToken(t *testing.T) {

	// isEndOfToken is a Parser method now, because hashtags and mentions terminate differently.
	// A default Parser carries the shared base set, which is what this test exercises.
	parser := New()

	require.False(t, parser.isEndOfToken('h', "hello world", 0))
	require.False(t, parser.isEndOfToken('e', "hello world", 1))
	require.False(t, parser.isEndOfToken('l', "hello world", 2))
	require.False(t, parser.isEndOfToken('l', "hello world", 3))
	require.False(t, parser.isEndOfToken('o', "hello world", 4))
	require.True(t, parser.isEndOfToken(' ', "hello world", 5))
	require.False(t, parser.isEndOfToken('w', "hello world", 6))
	require.False(t, parser.isEndOfToken('o', "hello world", 7))
	require.False(t, parser.isEndOfToken('r', "hello world", 8))
	require.False(t, parser.isEndOfToken('l', "hello world", 9))
	require.False(t, parser.isEndOfToken('d', "hello world", 10))
}
