package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOAuthUserTokenRequest_JSONBody confirms a JSON token request with Mastodon's snake_case names fills every field
func TestOAuthUserTokenRequest_JSONBody(t *testing.T) {

	// The body Ice Cubes posts to /oauth/token
	body := `{"grant_type":"authorization_code","client_id":"client-1","client_secret":"secret-1","redirect_uri":"icecubesapp://","code":"code-1","scope":"read write","refresh_token":"refresh-1","code_verifier":"verifier-1"}`

	request := NewOAuthUserTokenRequest()
	require.NoError(t, json.Unmarshal([]byte(body), &request))
	require.Equal(t, "authorization_code", request.GrantType)
	require.Equal(t, "client-1", request.ClientID)
	require.Equal(t, "secret-1", request.ClientSecret)
	require.Equal(t, "icecubesapp://", request.RedirectURI)
	require.Equal(t, "code-1", request.Code)
	require.Equal(t, "read write", request.Scope)
	require.Equal(t, "refresh-1", request.RefreshToken)
	require.Equal(t, "verifier-1", request.CodeVerifier)
}

// TestOAuthUserTokenRevokeRequest_JSONBody confirms a JSON revoke request with Mastodon's snake_case names fills every field
func TestOAuthUserTokenRevokeRequest_JSONBody(t *testing.T) {

	request := NewOAuthUserTokenRevokeRequest()
	require.NoError(t, json.Unmarshal([]byte(`{"client_id":"client-1","client_secret":"secret-1","token":"token-1"}`), &request))
	require.Equal(t, "client-1", request.ClientID)
	require.Equal(t, "secret-1", request.ClientSecret)
	require.Equal(t, "token-1", request.Token)
}
