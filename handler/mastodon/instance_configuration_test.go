package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestInstanceConfiguration confirms every limit a client requires is set, and that v1 reports the same values as v2.
func TestInstanceConfiguration(t *testing.T) {

	v2 := instanceConfiguration()

	// Clients such as Ice Cubes refuse a server whose statuses or polls block is missing or zero
	require.Positive(t, v2.Statuses.MaxCharacters)
	require.Positive(t, v2.Statuses.MaxMediaAttachments)
	require.Positive(t, v2.Statuses.CharactersReservedPerURL)
	require.NotEmpty(t, v2.MediaAttachments.SupportedMimeTypes)
	require.Positive(t, v2.MediaAttachments.ImageSizeLimit)
	require.Positive(t, v2.MediaAttachments.VideoSizeLimit)
	require.Positive(t, v2.Polls.MaxOptions)
	require.Positive(t, v2.Polls.MaxCharactersPerOption)
	require.Less(t, v2.Polls.MinExpiration, v2.Polls.MaxExpiration)
	require.False(t, v2.Translation.Enabled)

	// The deprecated v1 shape carries the same numbers
	v1 := instanceConfigurationV1()
	require.Equal(t, v2.Statuses.MaxCharacters, v1.Statuses.MaxCharacters)
	require.Equal(t, v2.Statuses.MaxMediaAttachments, v1.Statuses.MaxMediaAttachments)
	require.Equal(t, v2.MediaAttachments.SupportedMimeTypes, v1.MediaAttachments.SupportedMimeTypes)
	require.Equal(t, v2.Polls.MaxOptions, v1.Polls.MaxOptions)
	require.Equal(t, v2.Polls.MaxExpiration, v1.Polls.MaxExpiration)
}

// TestInstanceThumbnail confirms a domain with no image still reports a thumbnail URL, since clients reject an empty one.
func TestInstanceThumbnail(t *testing.T) {

	domain := model.Domain{Hostname: "example.com"}
	require.NotEmpty(t, domain.ImageURL())
	require.Contains(t, domain.ImageURL(), "example.com")
}
