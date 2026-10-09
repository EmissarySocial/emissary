package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/tools/secretcheck"
	mockdb "github.com/benpate/data-mock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// TestJWT_ParseErrorsOmitTheToken confirms a token that fails to parse or verify never appears in the error
func TestJWT_ParseErrorsOmitTheToken(t *testing.T) {

	service := NewJWT()
	service.Refresh(mockdb.New())

	tokens := map[string]string{
		"malformed":      "eyJhbGciOiJIUzUxMiJ9.bm90LWpzb24tc2VjcmV0LXBheWxvYWQ.c2lnbmF0dXJl",
		"wrongSignature": wronglySignedToken(t, &service),
	}

	for name, token := range tokens {

		t.Run(name+"/ParseString", func(t *testing.T) {
			_, err := service.ParseString(token)
			require.Error(t, err)
			secretcheck.RequireAbsent(t, err, token)
		})

		t.Run(name+"/ParseToken", func(t *testing.T) {
			err := service.ParseToken(token, jwt.MapClaims{})
			require.Error(t, err)
			secretcheck.RequireAbsent(t, err, token)
		})
	}
}

// wronglySignedToken returns a well-formed token that names the service's current key but is signed with another
func wronglySignedToken(t *testing.T, service *JWT) string {

	t.Helper()

	keyName, _, err := service.GetCurrentKey()
	require.NoError(t, err)

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{"sub": "someone"})
	token.Header["kid"] = keyName

	result, err := token.SignedString([]byte("not-the-service-key"))
	require.NoError(t, err)

	return result
}
