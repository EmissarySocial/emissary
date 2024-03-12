package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/builder"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

func GetStartup(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetStartup"

	return func(ctx echo.Context) error {

		// Get the factory for this Domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error finding domain")
		}

		// Authenticate the page request
		sterankoContext := ctx.(*steranko.Context)

		// Only domain owners can access admin pages
		if !isOwner(sterankoContext.Authorization()) {
			return derp.NewUnauthorizedError(location, "Unauthorized")
		}

		// Collect parameters to build
		templateService := factory.Template()
		template, err := templateService.LoadAdmin("startup")

		if err != nil {
			return derp.Wrap(err, location, "Error loading template")
		}

		actionID := first.String(ctx.Param("action"), "page")

		// Get a Builder for this page (also authenticates admin permissions)
		builder, err := builder.NewDomain(factory, ctx.Request(), ctx.Response(), template, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating builder")
		}

		// Render the HTML page.
		result, err := builder.Render()

		if err != nil {
			return derp.Wrap(err, location, "Error building page")
		}

		// Return the HTML page to the browser
		return ctx.HTML(http.StatusOK, string(result))
	}
}

func PostStartup(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.PostStartup"

	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error finding domain")
		}

		// Authenticate the page request
		sterankoContext := ctx.(*steranko.Context)

		// Only domain owners can access admin pages
		if !isOwner(sterankoContext.Authorization()) {
			return derp.NewUnauthorizedError(location, "Unauthorized")
		}

		// Try to load the requested theme from the Theme Service
		themeService := factory.Theme()
		themeID := ctx.QueryParam("themeId")
		theme := themeService.GetTheme(themeID)

		if theme.IsEmpty() {
			return derp.NewNotFoundError("handler.PostStartup", "Theme not found", themeID)
		}

		// Load/Initialize the Domain value
		domainService := factory.Domain()
		domain, err := domainService.LoadOrCreateDomain()

		if err != nil {
			return derp.Wrap(err, location, "Error loading domain")
		}

		// Save the new ThemeID to the database
		domain.ThemeID = themeID
		if err := domainService.Save(domain, "Change Theme"); err != nil {
			return derp.Wrap(err, location, "Error saving domain", domain)
		}

		// Initialize Streams
		streamService := factory.Stream()
		if err := streamService.Startup(&theme); err != nil {
			return derp.Wrap(err, location, "Error initializing Streams", themeID)
		}

		// Initialize Groups
		groupService := factory.Group()
		if err := groupService.Startup(&theme); err != nil {
			return derp.Wrap(err, location, "Error initializing Groups", themeID)
		}

		// Success!!!!!
		ctx.Response().Header().Set("HX-Redirect", "/home")
		return ctx.NoContent(http.StatusOK)
	}
}
