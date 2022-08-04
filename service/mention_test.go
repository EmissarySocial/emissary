package service

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

func TestWebMentionVerify(t *testing.T) {

	service := NewMention(nil)

	require.Nil(t, service.Verify("https://www.wikipedia.org", "https://meta.wikimedia.org/wiki/Privacy_policy", nil))
	require.NotNil(t, service.Verify("https://www.wikipedia.org", "https://non-existent.link.com", nil))
	require.Equal(t, 500, derp.ErrorCode(service.Verify("https://unavailable.thiswebsitedoesntexists824723834837.com", "", nil)))
}

func TestWebMentionDiscover_WebMentions(t *testing.T) {

	service := NewMention(nil)

	endpoint, err := service.DiscoverEndpoint("https://webmention.io")
	require.Equal(t, "https://webmention.io/pingback/webmention", endpoint)
	require.Nil(t, err)
}

func TestWebMentionDiscover_Wikipedia(t *testing.T) {

	service := NewMention(nil)

	endpoint, err := service.DiscoverEndpoint("https://www.wikipedia.org")
	require.Empty(t, endpoint)
	require.Equal(t, 400, derp.ErrorCode(err))
}
