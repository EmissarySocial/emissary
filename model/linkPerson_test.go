package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

func TestPersonLink(t *testing.T) {

	person := NewPersonLink()
	s := schema.New(PersonLinkSchema())

	require.Nil(t, s.Set(&person, "internalId", "000000000000000000000001"))
	require.Nil(t, s.Set(&person, "name", "TEST-PERSON"))
	require.Nil(t, s.Set(&person, "profileUrl", "http://profile.url"))
	require.Nil(t, s.Set(&person, "inboxUrl", "http://inbox.url.url"))
	require.Nil(t, s.Set(&person, "imageUrl", "https://image.url/with/path"))
	require.Nil(t, s.Set(&person, "emailAddress", "test@person.url"))
	require.NotNil(t, s.Set(&person, "missing", "missing"))

	{
		value, err := s.Get(&person, "internalId")
		require.Nil(t, err)
		require.Equal(t, "000000000000000000000001", value)
	}

	{
		value, err := s.Get(&person, "name")
		require.Nil(t, err)
		require.Equal(t, "TEST-PERSON", value)
	}

	{
		value, err := s.Get(&person, "profileUrl")
		require.Nil(t, err)
		require.Equal(t, "http://profile.url", value)
	}

	{
		value, err := s.Get(&person, "inboxUrl")
		require.Nil(t, err)
		require.Equal(t, "http://inbox.url.url", value)
	}

	{
		value, err := s.Get(&person, "imageUrl")
		require.Nil(t, err)
		require.Equal(t, "https://image.url/with/path", value)
	}

	{
		value, err := s.Get(&person, "emailAddress")
		require.Nil(t, err)
		require.Equal(t, "test@person.url", value)
	}

	{
		_, err := s.Get(&person, "missing")
		require.NotNil(t, err)
	}
}
