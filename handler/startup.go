package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
)

func GetStartup(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetStartup"

	// Only domain owners can access admin pages
	if !isOwner(ctx.Authorization()) {
		return derp.UnauthorizedError(location, "Unauthorized")
	}

	// Collect parameters to build
	templateService := factory.Template()
	template, err := templateService.LoadAdmin("startup")

	if err != nil {
		return derp.Wrap(err, location, "Unable to load template")
	}

	actionID := first.String(ctx.Param("action"), "page")

	// Get a Builder for this page (also authenticates admin permissions)
	builder, err := build.NewDomain(factory, session, ctx.Request(), ctx.Response(), template, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to create builder")
	}

	// Render the HTML page.
	result, err := builder.Render()

	if err != nil {
		return derp.Wrap(err, location, "Error building page")
	}

	// Return the HTML page to the browser
	return ctx.HTML(http.StatusOK, string(result))
}

func PostStartup(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostStartup"

	// Only domain owners can access admin pages
	if !isOwner(ctx.Authorization()) {
		return derp.UnauthorizedError(location, "Unauthorized")
	}

	// Try to load the requested theme from the Theme Service
	themeService := factory.Theme()
	themeID := ctx.QueryParam("themeId")
	theme := themeService.GetTheme(themeID)

	if theme.IsEmpty() {
		return derp.NotFoundError("handler.PostStartup", "Theme not found", themeID)
	}

	// Load/Initialize the Domain value
	domainService := factory.Domain()
	domain := *domainService.Get()

	// Save the new ThemeID to the database
	domain.ThemeID = themeID
	if err := domainService.Save(session, domain, "Change Theme"); err != nil {
		return derp.Wrap(err, location, "Unable to save domain", domain)
	}

	// Initialize Streams
	streamService := factory.Stream()
	if err := streamService.Startup(session, &theme); err != nil {
		return derp.Wrap(err, location, "Unable to initialize Streams", themeID)
	}

	// Initialize Groups
	groupService := factory.Group()
	if err := groupService.Startup(session, &theme); err != nil {
		return derp.Wrap(err, location, "Unable to initialize Groups", themeID)
	}

	// Success!!!!!
	ctx.Response().Header().Set("HX-Redirect", "/home")
	return ctx.NoContent(http.StatusOK)
}
