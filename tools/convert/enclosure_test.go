package convert

import (
	"bytes"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/gorilla/feeds"
	"github.com/stretchr/testify/require"
)

// TestStreamEnclosure verifies that a Stream offers an enclosure only when it names one, and that its
// type defaults to MP3 audio
func TestStreamEnclosure(t *testing.T) {

	stream := model.NewStream()
	require.Nil(t, StreamToGorillaFeed(stream).Enclosure)
	require.Nil(t, StreamToJsonFeed(stream).Attachments)

	stream.Data["enclosureUrl"] = "https://example.com/episode/audio?format=mp3"
	require.Equal(t, &feeds.Enclosure{Url: "https://example.com/episode/audio?format=mp3", Type: "audio/mpeg", Length: "0"}, StreamToGorillaFeed(stream).Enclosure)

	attachments := StreamToJsonFeed(stream).Attachments
	require.Len(t, attachments, 1)
	require.Equal(t, "audio/mpeg", attachments[0].MIMEType)

	stream.Data["enclosureType"] = "audio/ogg"
	require.Equal(t, "audio/ogg", StreamToGorillaFeed(stream).Enclosure.Type)
}

// TestStreamEnclosure_RSS verifies that the enclosure reaches the RSS document, with the URL's query
// string escaped
func TestStreamEnclosure_RSS(t *testing.T) {

	stream := model.NewStream()
	stream.Label = "Episode 1"
	stream.Data["enclosureUrl"] = "https://example.com/episode/audio?format=mp3&x=1"

	feed := feeds.Feed{Title: "Podcast", Link: &feeds.Link{Href: "https://example.com"}}
	feed.Items = []*feeds.Item{StreamToGorillaFeed(stream)}

	var buffer bytes.Buffer
	require.NoError(t, feed.WriteRss(&buffer))
	require.Contains(t, buffer.String(), `<enclosure url="https://example.com/episode/audio?format=mp3&amp;x=1" length="0" type="audio/mpeg"></enclosure>`)
}
