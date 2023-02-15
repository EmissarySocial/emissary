package render

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	accept "github.com/timewasted/go-accept-headers"
)

func TestWebSubAccept(t *testing.T) {

	{
		mimeStack := "application/feed+json; q=1.0, application/json; q=0.9, application/atom+xml; q=0.8, application/rss+xml; q=0.7, application/xml; q=0.6, text/xml; q=0.5, text/html; q=0.4, */*; q=0.1"
		format, err := accept.Negotiate(mimeStack, model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText)

		require.Nil(t, err)
		require.Equal(t, "application/feed+json", format)
	}

	{
		mimeStack := "application/json; q=0.9, application/atom+xml; q=0.8, application/rss+xml; q=0.7, application/xml; q=0.6, text/xml; q=0.5, text/html; q=0.4, */*; q=0.1"
		format, err := accept.Negotiate(mimeStack, model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText, model.MimeTypeJSON)

		require.Nil(t, err)
		require.Equal(t, "application/json", format)
	}

	{
		mimeStack := "application/atom+xml; q=0.8, application/rss+xml; q=0.7, application/xml; q=0.6, text/xml; q=0.5, text/html; q=0.4, */*; q=0.1"
		format, err := accept.Negotiate(mimeStack, model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText)

		require.Nil(t, err)
		require.Equal(t, "application/atom+xml", format)
	}
}

func TestAcceptFailure(t *testing.T) {

	{
		mimeStack := "application/atom+xml; q=0.8, application/rss+xml; q=0.7, application/xml; q=0.6, text/xml; q=0.5, */*; q=0.1"
		format, err := accept.Negotiate(mimeStack, model.MimeTypeHTML)

		require.Nil(t, err)
		require.Equal(t, "application/atom+xml", format)
	}
}
