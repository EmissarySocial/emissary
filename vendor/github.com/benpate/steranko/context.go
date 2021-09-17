package steranko

import (
	"github.com/benpate/derp"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

// Context extends the echo context with an authenticated JWT Token.
type Context struct {
	steranko *Steranko
	claims   jwt.Claims
	echo.Context
}

func (ctx *Context) Authorization() (jwt.Claims, error) {

	// Only comput this once, then store in the context for next time.
	if ctx.claims == nil {

		// Retrieve the cookie value from the context
		name := cookieName(ctx)
		tokenString, err := ctx.Cookie(name)

		if err != nil {
			return nil, derp.Wrap(err, "steranko.Context.Claims", "Invalid cookie")
		}

		claims := ctx.steranko.UserService.NewClaims()

		// Parse it as a JWT token
		token, err := jwt.ParseWithClaims(tokenString.Value, claims, ctx.steranko.KeyService.FindJWTKey)

		if err != nil {
			return nil, derp.Wrap(err, "steranko.Context.Claims", "Error parsing token")
		}

		if !token.Valid {
			return nil, derp.New(derp.CodeForbiddenError, "steranko.Context.Claims", "Invalid token")
		}

		// Save this value in the context for next time.
		ctx.claims = claims
	}

	return ctx.claims, nil
}
