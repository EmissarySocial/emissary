package handler

import (
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// accessTokenParser decodes a JWT's claims without checking its signature
var accessTokenParser = jwt.NewParser()

// withoutAccessTokens detaches every OAuth access token from the request before the next handler reads it
func withoutAccessTokens(next echo.HandlerFunc) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		detachAccessTokens(ctx.Request())
		return next(ctx)
	}
}

// detachAccessTokens removes every OAuth access token from a request's Authorization header and
// cookies, so the session machinery sees the request as anonymous
func detachAccessTokens(request *http.Request) {

	// The Authorization header carries one credential, with or without the "Bearer" scheme
	if isAccessToken(bearerToken(request.Header.Get("Authorization"))) {
		request.Header.Del("Authorization")
	}

	// Keep every cookie that is not an access token. Cookies are judged by value, never by name,
	// so the active session, its masquerade backup, and the plain-HTTP name are all covered.
	cookies := request.Cookies()
	kept := make([]*http.Cookie, 0, len(cookies))

	for _, cookie := range cookies {
		if !isAccessToken(cookie.Value) {
			kept = append(kept, cookie)
		}
	}

	// Rewrite the Cookie header only when something was removed, so every other request passes untouched
	if len(kept) == len(cookies) {
		return
	}

	request.Header.Del("Cookie")

	for _, cookie := range kept {
		request.AddCookie(cookie)
	}
}

// bearerToken returns the credential in an Authorization header, with any "Bearer" scheme removed
func bearerToken(header string) string {

	scheme, token, found := strings.Cut(header, " ")

	// A header with no scheme is a bare token
	if !found {
		return header
	}

	// Another scheme is left whole, so it fails to decode as a JWT
	if !strings.EqualFold(scheme, "Bearer") {
		return header
	}

	return strings.TrimSpace(token)
}

// isAccessToken returns TRUE if a JWT's claims mark it as an OAuth access token
func isAccessToken(tokenString string) bool {

	if tokenString == "" {
		return false
	}

	// The signature is not checked, because the answer only decides whether the credential is removed,
	// and a removed credential grants nothing. A token left in place still meets steranko's signature check.
	claims := model.NewAuthorization()

	if _, _, err := accessTokenParser.ParseUnverified(tokenString, &claims); err != nil {
		return false
	}

	return claims.IsAccessToken()
}
