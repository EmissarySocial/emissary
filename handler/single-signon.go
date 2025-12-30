package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
)

func GetSingleSignOn(ctx *steranko.Context, factory *service.Factory, session data.Session, domain *model.Domain) error {
	const location = "handler.GetSingleSignOn"

	// RULE: Guarantee that the SSO is active
	if domain.Data.GetString("sso_active") != "true" {
		return derp.NotFound(location, "Single Sign-On is not active")
	}

	// RULE: Guarantee that the SSO secret has been set
	secret := domain.Data.GetString("sso_secret")

	if secret == "" {
		return derp.Internal(location, "SSO secret key is not set")
	}

	// Parse the JWT Token
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}

	tokenString := ctx.QueryParam("token")
	claims := jwt.MapClaims{}

	if _, err := jwt.ParseWithClaims(tokenString, &claims, keyFunc, steranko.JWTValidMethods()); err != nil {
		return derp.BadRequest(location, "Invalid JWT token")
	}

	// Extract User Information from the Token
	username := convert.String(claims["username"])

	// Look up the user in the database.
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByUsername(session, username, &user); err != nil {
		return derp.Wrap(err, location, "Unable to load user")
	}

	// Create a sign-in session for the user
	if err := factory.Steranko(session).SigninUser(ctx, &user); err != nil {
		return derp.Wrap(err, location, "Unable to create certificate")
	}

	// Forward to the user's profile page
	return ctx.Redirect(http.StatusSeeOther, "/@"+user.Username)
}
