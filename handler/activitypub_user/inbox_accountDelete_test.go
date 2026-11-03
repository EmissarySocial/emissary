package activitypub_user

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsAccountSelfDelete covers both object shapes, a Delete of something else, and other activities
func TestIsAccountSelfDelete(t *testing.T) {

	require.True(t, isAccountSelfDelete([]byte(`{"type":"Delete","actor":"https://a.example/u/1","object":"https://a.example/u/1"}`)))
	require.True(t, isAccountSelfDelete([]byte(`{"type":"Delete","actor":"https://a.example/u/1","object":{"id":"https://a.example/u/1","type":"Person"}}`)))
	require.False(t, isAccountSelfDelete([]byte(`{"type":"Delete","actor":"https://a.example/u/1","object":"https://a.example/notes/2"}`)), "deleting a post is not an account deletion")
	require.False(t, isAccountSelfDelete([]byte(`{"type":"Create","actor":"https://a.example/u/1","object":"https://a.example/u/1"}`)))
	require.False(t, isAccountSelfDelete([]byte(`{"type":"Delete","object":""}`)))
	require.False(t, isAccountSelfDelete([]byte(`not json`)))
	require.False(t, isAccountSelfDelete(nil))
}

// TestPeekRequestBody confirms the request still reads its whole body after peeking, even past the limit
func TestPeekRequestBody(t *testing.T) {

	request := httptest.NewRequest("POST", "/inbox", strings.NewReader("0123456789"))

	peeked := peekRequestBody(request, 4)
	require.Equal(t, "0123", string(peeked))

	rest, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	require.Equal(t, "0123456789", string(rest))
}
