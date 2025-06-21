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
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	// Collect parameters from the request
	identityService := factory.Identity()
	identifier := ctx.FormValue("identifier")
	identifierType := identityService.GuessIdentifierType(identifier)

	if identifierType == "" {
		return derp.BadRequestError(location, "Unrecognized Identifier Type", identifierType)
	}

	// Create and send a guest signin code
	if err := identityService.SendGuestCode(nil, identifierType, identifier); err != nil {

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

	// Get the Authorization from the Request
	authorization := getAuthorization(ctx)

	// Read and parse the JWT token
	tokenString := ctx.Param("jwt")
	jwtService := factory.JWT()

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, jwtService.FindKey, steranko.JWTValidMethods())

	if err != nil {
		return derp.Wrap(err, location, "Error parsing JWT Token")
	}

	if !token.Valid {
		return derp.Wrap(err, location, "Invalid JWT Token")
	}

	// Collect the identifier and identifier type from the JWT claims
	identityService := factory.Identity()
	identifier := convert.String(claims["A"])
	identifierType := convert.String(claims["T"])

	// If the JWT token has an IdentityID, then set this in the Authorization
	// (overriding a pre-existing IdentityID, if any)
	if identityID, exists := claims["I"]; exists {
		identityIDstring := convert.String(identityID)
		if identityID, err := primitive.ObjectIDFromHex(identityIDstring); err == nil {
			authorization.IdentityID = identityID
		}
	}

	switch authorization.IdentityID {

	// If the JWT is not already linked to an Identity, then load or create one from scratch.
	case primitive.NilObjectID:

		identity, err := identityService.LoadOrCreate("", identifierType, identifier)

		if err != nil {
			return derp.InternalError(location, "Error loading/creating new Identity", identifier)
		}

		// Update the Authorization with the (new?) IdentityID
		authorization.IdentityID = identity.IdentityID

		// Create a new JWT token and return it as a cookie
		if err := factory.Steranko().SetCookie(ctx, authorization); err != nil {
			return derp.Wrap(err, location, "Error setting authorization cookie")
		}

	// Otherwise, add/update the identifier in the existing Identity
	default:

		identity := model.NewIdentity()

		if err := identityService.LoadByID(authorization.IdentityID, &identity); err != nil {
			return derp.Wrap(err, location, "Error loading Identity by ID", authorization.IdentityID)
		}

		identity.SetIdentifier(identifierType, identifier)

		if err := identityService.Save(&identity, "Added/Updated Identifier: "+identifierType); err != nil {
			return derp.Wrap(err, location, "Error saving Identity with new identifier", identity.IdentityID)
		}
	}

	return ctx.Redirect(http.StatusSeeOther, "/@guest")
}

// PostIdentityIdentifier allows guests to edit the identifiers for their Identity.
func PostIdentityIdentifier(ctx *steranko.Context, factory *domain.Factory, identity *model.Identity) error {

	const location = "handler.PostIdentityEditIdentifier"
	// Get the identifier type and value from the request
	identifierType := ctx.FormValue("identifierType")
	identifierValue := ctx.FormValue("identifier")
	identityService := factory.Identity()

	// If we're setting a new identifier, then send a guest code to the user
	if identifierValue != "" {

		if err := identityService.SendGuestCode(identity, identifierType, identifierValue); err != nil {
			return derp.Wrap(err, location, "Error setting identifier on Identity")
		}

		return ctx.Redirect(http.StatusSeeOther, "/@guest/confirm")
	}

	// Fall through means we're deleting an existing identifier
	identity.SetIdentifier(identifierType, "")

	if err := identityService.Save(identity, "Removed identifier: "+identifierType); err != nil {
		return derp.Wrap(err, location, "Error saving Identity", identity.IdentityID)
	}

	return closeModalAndRefreshPage(ctx)
}
