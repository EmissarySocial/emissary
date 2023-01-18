package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

func TestDocumentLink(t *testing.T) {

	document := NewDocumentLink()
	s := schema.New(DocumentLinkSchema())

	require.Nil(t, s.Set(&document, "internalId", "000000000000000000000001"))
	require.Nil(t, s.Set(&document, "author.name", "PERSON"))
	require.Nil(t, s.Set(&document, "url", "http://document.url"))
	require.Nil(t, s.Set(&document, "type", "TYPE"))
	require.Nil(t, s.Set(&document, "label", "LABEL"))
	require.Nil(t, s.Set(&document, "summary", "SUMMARY"))
	require.Nil(t, s.Set(&document, "imageUrl", "http://image.url"))
	require.Nil(t, s.Set(&document, "publishDate", "1"))
	require.Nil(t, s.Set(&document, "updateDate", "2"))
	require.NotNil(t, s.Set(&document, "missing", "missing"))

	{
		value, err := s.Get(&document, "internalId")
		require.Nil(t, err)
		require.Equal(t, "000000000000000000000001", value)
	}

	{
		value, err := s.Get(&document, "author.name")
		require.Nil(t, err)
		require.Equal(t, "PERSON", value)
	}

	{
		value, err := s.Get(&document, "url")
		require.Nil(t, err)
		require.Equal(t, "http://document.url", value)
	}

	{
		value, err := s.Get(&document, "type")
		require.Nil(t, err)
		require.Equal(t, "TYPE", value)
	}

	{
		value, err := s.Get(&document, "label")
		require.Nil(t, err)
		require.Equal(t, "LABEL", value)
	}

	{
		value, err := s.Get(&document, "summary")
		require.Nil(t, err)
		require.Equal(t, "SUMMARY", value)
	}

	{
		value, err := s.Get(&document, "imageUrl")
		require.Nil(t, err)
		require.Equal(t, "http://image.url", value)
	}

	{
		value, err := s.Get(&document, "publishDate")
		require.Nil(t, err)
		require.Equal(t, int64(1), value)
	}

	{
		value, err := s.Get(&document, "updateDate")
		require.Nil(t, err)
		require.Equal(t, int64(2), value)
	}

	{
		_, err := s.Get(&document, "missing")
		require.NotNil(t, err)
	}
}
