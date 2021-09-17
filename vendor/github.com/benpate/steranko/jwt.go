package steranko

import (
	"github.com/benpate/derp"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

// createJWT creates a new JWT token for the provided user.
// TODO: include additional configuration options when defined.
func (s *Steranko) createJWT(user User) (string, error) {

	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = user.Claims()

	keyID, key := s.KeyService.NewJWTKey()

	token.Header["kid"] = keyID

	// Generate encoded token and send it as response.
	signedString, errr := token.SignedString(key)

	if errr != nil {
		return "", derp.Wrap(errr, "steranko.PostSigninTransaction", "Error Signing JWT Token")
	}

	return signedString, nil
}

// cookieName returns the correct cookie name to use, based on the kind of connection.
// If connecting via HTTP, then "Authorization" is used.
// If connecting via SSL, then "__Host-Authorization" is used so that the cookie is "domain locked".  See [https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#cookie_prefixes]
func cookieName(ctx echo.Context) string {

	// If this is a secure domain...
	if ctx.IsTLS() {
		// Use a cookie name that can only be set on an SSL connection, and is "domain-locked"
		return "__Host-Authorization"
	}

	return "Authorization"
}
