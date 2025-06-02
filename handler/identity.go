package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
)

// GetIdentity handles GET request for the /@guest route
func GetIdentity(ctx *steranko.Context, factory *domain.Factory, identity *model.Identity) error {
	return buildIdentity(ctx, factory, identity, build.ActionMethodGet)
}

// PostIdentity handles POST request for the /@guest route
func PostIdentity(ctx *steranko.Context, factory *domain.Factory, identity *model.Identity) error {
	return buildIdentity(ctx, factory, identity, build.ActionMethodPost)
}

// buildIdentity is the common function that handles both GET and POSt requests for the /@guest route.
func buildIdentity(ctx *steranko.Context, factory *domain.Factory, identity *model.Identity, actionMethod build.ActionMethod) error {
	const location = "handler.GetIdentity"

	// Create a builder
	actionID := getActionID(ctx)
	builder, err := build.NewIdentity(factory, ctx.Request(), ctx.Response(), identity, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Error creating builder")
	}

	// Build the HTML response
	if err := build.AsHTML(factory, ctx, builder, actionMethod); err != nil {
		return derp.Wrap(err, location, "Error building page")
	}

	return ctx.NoContent(http.StatusOK)
}

// GetIdentitySignin displays the Signin page for guest Identities
func GetIdentitySignin(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.GetIdentitySignin"

	// Get the standard Signin page
	template := factory.Domain().Theme().HTMLTemplate

	domain := factory.Domain().Get()

	// Get a clean version of the URL query parameters
	data := cleanQueryParams(ctx.QueryParams())
	data["domainName"] = domain.Label
	data["domainIcon"] = domain.IconURL()

	// Render the template
	if err := template.ExecuteTemplate(ctx.Response(), "guest-signin", data); err != nil {
		return derp.Wrap(err, location, "Error executing template")
	}

	return ctx.JSON(http.StatusOK, "")
}

// PostIdentitySignin accepts POST from the guest Signin page, and sends
// guest signin codes to the email/handle provided by the user.
func PostIdentitySignin(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.PostIdentitySignin"

	// Create and send a guest signin code
	identityService := factory.Identity()
	if err := identityService.SendGuestCode(ctx.FormValue("identifier")); err != nil {

		// Report the error for debugging...
		derp.Report(derp.Wrap(err, location, "Error sending Guest Code"))

		// Report errors to the caller
		return inlineError(ctx, "Can't send guest code. Please double check your address.")
	}

	// Write a response to the caller
	b := html.New()
	b.H1().InnerText("Please check your inbox").Close()
	b.Div().InnerText("Your signin code should be delivered to your inbox in just a moment. Please click the link you find there to sign in.").Close()
	ctx.Response().Header().Set("Hx-Retarget", "#response")

	// Done!
	return ctx.HTML(http.StatusOK, b.String())
}

// GetIdentitySigninWithJWT receives JWT tokens from the request, and signs guests
// into the website with their Identity.
func GetIdentitySigninWithJWT(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.GetIdentitySigninWithJWT"

	// Read and parse the JWT token
	tokenString := ctx.Param("jwt")
	jwtService := factory.JWT()

	claims := jwt.MapClaims{}
	encryptionMethods := []string{"HS256", "HS384", "HS512"}
	token, err := jwt.ParseWithClaims(tokenString, &claims, jwtService.FindKey, jwt.WithValidMethods(encryptionMethods))

	if err != nil {
		return derp.Wrap(err, location, "Error parsing JWT Token")
	}

	if !token.Valid {
		return derp.Wrap(err, location, "Invalid JWT Token")
	}

	// Isolate the Identifier
	identityService := factory.Identity()
	identifier := convert.String(claims["id"])
	identifierType := identityService.GuessIdentifierType(identifier)
	identity, err := identityService.LoadOrCreate(identifierType, identifier, true)

	if err != nil {
		return derp.InternalError(location, "Error loading/creating new Identity", identifier)
	}

	// Update the Authorization with the (new?) IdentityID
	authorization := getAuthorization(ctx)
	authorization.IdentityID = identity.IdentityID

	// Create a new JWT token and return it as a cookie
	steranko := factory.Steranko()
	if err := steranko.SetCookieFromClaims(ctx, authorization); err != nil {
		return derp.Wrap(err, location, "Error setting authorization cookie")
	}

	return ctx.Redirect(http.StatusSeeOther, "/@guest")
}
