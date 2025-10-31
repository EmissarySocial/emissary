package handler

import (
	"io"
	"io/fs"
	"mime"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/labstack/echo/v4"
)

func GetThemeResource(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		themeID := ctx.Param("themeId")
		filename := ctx.Param("filename")

		themeService := serverFactory.Theme()
		theme := themeService.GetTheme(themeID)

		return getResource(theme.Resources, filename, ctx.Response())
	}
}

func GetTemplateResource(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		templateID := ctx.Param("templateId")
		filename := ctx.Param("filename")

		templateService := serverFactory.Template()
		template, err := templateService.Load(templateID)

		if err != nil {
			return derp.NotFoundError("handler.GetTemplateResource", "Template not found", templateID)
		}

		return getResource(template.Resources, filename, ctx.Response())
	}
}

func GetWidgetResource(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		widgetID := ctx.Param("widgetId")
		filename := ctx.Param("filename")

		widgetService := serverFactory.Widget()
		widget, ok := widgetService.Get(widgetID)

		if !ok {
			return derp.NotFoundError("handler.GetWidgetResource", "Widget not found", widgetID)
		}

		return getResource(widget.Resources, filename, ctx.Response())
	}
}

func getResource(filesystem fs.FS, filename string, response *echo.Response) error {

	const location = "handler.getResource"

	// Guarantee that this filesystem is not empty
	if filesystem == nil {
		return derp.NotFoundError(location, "Resource not found", filename)
	}

	// Try to open the file from the filesystem
	file, err := filesystem.Open(filename)

	if err != nil {
		return derp.NotFound(location, "Unable to open resource", filename, err.Error())
	}

	defer derp.ReportFunc(file.Close)

	// Prepare response headers
	extension := "." + list.Last(filename, '.')
	contentType := mime.TypeByExtension(extension)
	cacheControl := "public, max-age=2592000"

	header := response.Header()
	header.Set("Content-Type", contentType)
	header.Set("Cache-Control", cacheControl)
	response.WriteHeader(200)

	// Write the file to the client
	if _, err := io.Copy(response, file); err != nil {
		return derp.Wrap(err, location, "Unable to write resource content to client", filename)
	}

	// [Station](https://billandted.fandom.com/wiki/Station)
	return nil
}
