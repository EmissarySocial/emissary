package parse

import (
	"testing"

	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

func TestSplit(t *testing.T) {

	require.Equal(t, sliceof.String{}, Split(""))
	require.Equal(t, sliceof.String{"hello", "world"}, Split("hello world"))
	require.Equal(t, sliceof.String{"hello", "world"}, Split("hello  world"))
	require.Equal(t, sliceof.String{"hello", "world"}, Split("#hello world"))
	require.Equal(t, sliceof.String{"hello", "world"}, Split("#hello #world"))
	require.Equal(t, sliceof.String{"hello", "world", "people"}, Split("hello, world,people "))
	require.Equal(t, sliceof.String{"hey", "there", "ladies", "and", "gentlemen"}, Split("hey there ladies and gentlemen"))
}

func TestIsEndOfToken(t *testing.T) {

	require.False(t, isEndOfToken('h', "hello world", 0))
	require.False(t, isEndOfToken('e', "hello world", 1))
	require.False(t, isEndOfToken('l', "hello world", 2))
	require.False(t, isEndOfToken('l', "hello world", 3))
	require.False(t, isEndOfToken('o', "hello world", 4))
	require.True(t, isEndOfToken(' ', "hello world", 5))
	require.False(t, isEndOfToken('w', "hello world", 6))
	require.False(t, isEndOfToken('o', "hello world", 7))
	require.False(t, isEndOfToken('r', "hello world", 8))
	require.False(t, isEndOfToken('l', "hello world", 9))
	require.False(t, isEndOfToken('d', "hello world", 10))
}
