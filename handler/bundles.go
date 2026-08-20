package handler

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
)

// ThemeProvider is the minimal server-factory surface needed to serve theme
// bundles and resources.  Both the live server and the setup console satisfy it.
type ThemeProvider interface {
	Theme() *service.Theme
}

// GetThemeBundle serves a bundled asset (CSS, JS, etc.) that belongs to a Theme
func GetThemeBundle(themeProvider ThemeProvider) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		themeID := ctx.Param("themeId")
		bundleID := ctx.Param("bundleId")

		themeService := themeProvider.Theme()
		theme := themeService.GetTheme(themeID)

		return getBundle(theme.Bundles, bundleID, ctx.Response())
	}
}

// GetTemplateBundle serves a bundled asset (CSS, JS, etc.) that belongs to a Template
func GetTemplateBundle(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		templateID := ctx.Param("templateId")
		bundleID := ctx.Param("bundleId")

		templateService := serverFactory.Template()
		template, err := templateService.Load(templateID)

		if err != nil {
			return derp.NotFound("handler.GetTemplateBundle", "Template not found", templateID)
		}

		return getBundle(template.Bundles, bundleID, ctx.Response())
	}
}

// GetWidgetBundle serves a bundled asset (CSS, JS, etc.) that belongs to a Widget
func GetWidgetBundle(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		widgetID := ctx.Param("widgetId")
		bundleID := ctx.Param("bundleId")

		widgetService := serverFactory.Widget()
		widget, ok := widgetService.Get(widgetID)

		if !ok {
			return derp.NotFound("handler.GetWidgetBundle", "Widget not found", widgetID)
		}

		return getBundle(widget.Bundles, bundleID, ctx.Response())
	}
}

// getBundle writes a single named Bundle to the response, with its content type and cache headers
func getBundle(bundles mapof.Object[model.Bundle], bundleID string, response *echo.Response) error {

	bundle, ok := bundles[bundleID]

	if !ok {
		return derp.NotFound("handler.getBundle", "Bundle not found", bundleID)
	}

	header := response.Header()
	header.Set("Content-Type", bundle.ContentType)
	header.Set("Cache-Control", bundle.GetCacheControl())
	response.WriteHeader(200)

	if _, err := response.Write(bundle.Content); err != nil {
		return derp.Wrap(err, "handler.getBundle", "Writing bundle content", bundleID)
	}

	return nil
}
