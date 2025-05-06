package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
)

func GetSingleSignOn(ctx *steranko.Context, factory *domain.Factory, domain *model.Domain) error {
	const location = "handler.GetSingleSignOn"

	// RULE: Guarantee that the SSO is active
	if domain.Data.GetString("sso_active") != "true" {
		return derp.NotFoundError(location, "Single Sign-On is not active")
	}

	// RULE: Guarantee that the SSO secret has been set
	secret := domain.Data.GetString("sso_secret")

	if secret == "" {
		return derp.InternalError(location, "SSO secret key is not set")
	}

	// Parse the JWT Token
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}

	tokenString := ctx.QueryParam("token")
	option := jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}) // https://pkg.go.dev/github.com/golang-jwt/jwt/v5@v5.2.1#WithValidMethods

	claims := jwt.MapClaims{}

	if _, err := jwt.ParseWithClaims(tokenString, &claims, keyFunc, option); err != nil {
		return derp.BadRequestError(location, "Invalid JWT token")
	}

	// Extract User Information from the Token
	username := convert.String(claims["username"])

	// Look up the user in the database.
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByUsername(username, &user); err != nil {
		return derp.Wrap(err, location, "Error loading user")
	}

	// Create a sign-in session for the user
	sterankoService := factory.Steranko()

	certificate, err := sterankoService.CreateCertificate(ctx.Request(), &user)

	if err != nil {
		return derp.Wrap(err, location, "Error creating certificate")
	}

	// Push the certificate and make a -backup cookie
	sterankoService.PushCookie(ctx, certificate)

	// Forward to the user's profile page
	return ctx.Redirect(http.StatusSeeOther, "/@"+user.Username)
}
