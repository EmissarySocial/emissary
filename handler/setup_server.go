package handler

import (
	_ "embed"
	"html/template"
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/table"
	"github.com/labstack/echo/v4"
)

// SetupPageGet generates simple template pages for the setup server, based on the Templates and ID provided.
func SetupPageGet(factory *server.Factory, templates *template.Template, templateID string) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		config := factory.Config()

		header := ctx.Response().Header()
		header.Set("Content-Type", model.MimeTypeHTML)
		header.Set("Cache-Control", "no-cache")

		if err := templates.ExecuteTemplate(ctx.Response().Writer, templateID, config); err != nil {
			derp.Report(build.WrapInlineError(ctx.Response(), derp.Wrap(err, "setup.getIndex", "Error building index page")))
		}

		return nil
	}
}

// SetupServerGet generates a form for the setup app.
func SetupServerGet(factory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Data schema and UI schema
		config := factory.Config()
		schema := config.Schema()
		section := ctx.Param("section")
		uri := "/server/" + section

		// Find the correct form for this section (or fail)
		element, asTable, err := getSetupForm(section)

		if err != nil {
			return derp.Wrap(err, "setup.serverTable", "Invalid table name")
		}

		// Write Table-formatted forms.
		if asTable {
			widget := table.New(&schema, &element, &config, section, factory.Icons(), uri)
			return widget.Draw(ctx.Request().URL, ctx.Response().Writer)
		}

		// Fall through to single form
		widget := form.New(schema, element)
		result, err := widget.Editor(&config, nil)

		if err != nil {
			return derp.Wrap(err, "setup.serverTable", "Error creating form")
		}

		// Return the form
		return ctx.HTML(http.StatusOK, build.WrapForm(uri, result, element.Encoding(), "cancel-button:hide"))
	}
}

// SetupServerPost saves the form data to the config file.
func SetupServerPost(factory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		data := mapof.NewAny()

		if err := ctx.Bind(&data); err != nil {
			return build.WrapInlineError(ctx.Response(), derp.Wrap(err, "setup.serverPost", "Error parsing form data"))
		}

		// Data schema and UI schema
		config := factory.Config()
		schema := config.Schema()
		section := ctx.Param("section")
		uri := "/server/" + section

		// Find the correct form for this section (or fail)
		element, asTable, err := getSetupForm(section)

		if err != nil {
			return build.WrapInlineError(ctx.Response(), derp.Wrap(err, "setup.serverTable", "Invalid table name"))
		}

		// Write Table-formatted forms.
		if asTable {
			widget := table.New(&schema, &element, &config, section, factory.Icons(), uri)

			// Apply the changes to the configuration
			if err := widget.Do(ctx.Request().URL, data); err != nil {
				return build.WrapInlineError(ctx.Response(), derp.Wrap(err, "setup.serverTable", "Error saving form data"))
			}

			// Try to save the configuration to the persistent storage
			if err := factory.UpdateConfig(config); err != nil {
				return build.WrapInlineError(ctx.Response(), derp.Wrap(err, "setup.postServer", "Internal error saving config.  Try again later."))
			}

			// Redraw the table
			return widget.DrawView(ctx.Response().Writer)
		}

		// Fall through to single form
		form := form.New(schema, element)

		// Apply the changes to the configuration
		if err := form.SetAll(&config, data, nil); err != nil {
			return build.WrapInlineError(ctx.Response(), derp.Wrap(err, "setup.serverPost", "Error saving form data", data))
		}

		// Try to save the configuration to the persistent storage
		if err := factory.UpdateConfig(config); err != nil {
			return build.WrapInlineError(ctx.Response(), derp.Wrap(err, "setup.postServer", "Internal error saving config.  Try again later."))
		}

		// Success!
		return build.WrapInlineSuccess(ctx.Response(), "Record Updated at: "+time.Now().Format(time.TimeOnly))
	}
}

// getSetupForm generates the different form layouts to use on the setup/server page.
func getSetupForm(name string) (form.Element, bool, error) {

	switch name {

	case "general":
		return form.Element{
			Type: "layout-group",
			Children: []form.Element{
				{Type: "layout-vertical", Label: "Common Services", Description: "Common queue and cache services used across multiple installations.  This should only be shared across trusted servers.", Children: []form.Element{
					{Type: "text", Label: "Database Connection String", Path: "activityPubCache.connectString", Description: "MongoDB connection string only"},
					{Type: "text", Label: "Database Name", Path: "activityPubCache.database"},
				}},
				{Type: "layout-vertical", Label: "Ports", Children: []form.Element{
					{Type: "text", Label: "HTTP", Description: "Port to use for HTTP connections (standard: 80, disabled: 0)", Path: "httpPort", Options: mapof.Any{"format": "number", "min": 0, "max:": 65535}},
					{Type: "text", Label: "HTTPS", Description: "Port to use for HTTPS connections (standard: 443, disabled: 0)", Path: "httpsPort", Options: mapof.Any{"format": "number", "min": 0, "max:": 65535}},
				}},
				{Type: "layout-vertical", Label: "Testing and Development", Children: []form.Element{
					{Type: "select", Label: "Debug Output", Path: "debugLevel"},
				}},
			},
		}, false, nil

	case "templates":
		return form.Element{
			Type: "layout-vertical",
			Children: []form.Element{
				{Type: "select", Label: "Adapter", Path: "adapter"},
				{Type: "text", Label: "Location", Path: "location", Options: mapof.Any{"column-width": "100%"}},
			},
		}, true, nil

	case "attachments":
		return form.Element{
			Type:        "layout-group",
			Description: "Readable/Writeable location where uploaded files (originals and thumbnails) are stored.",
			Children: []form.Element{
				{Type: "layout-vertical", Label: "Originals", Children: []form.Element{
					{Type: "select", Label: "Adapter", Path: "attachmentOriginals.adapter"},
					{Type: "text", Label: "Location / Endpoint", Path: "attachmentOriginals.location"},
					{Type: "text", Label: "AccessKey", Path: "attachmentOriginals.accessKey", Options: mapof.Any{"show-if": "attachmentOriginals.adapter eq S3"}},
					{Type: "text", Label: "SecretKey", Path: "attachmentOriginals.secretKey", Options: mapof.Any{"show-if": "attachmentOriginals.adapter eq S3"}},
					{Type: "text", Label: "Token", Path: "attachmentOriginals.token", Options: mapof.Any{"show-if": "attachmentOriginals.adapter eq S3"}},
					{Type: "text", Label: "Region", Path: "attachmentOriginals.region", Options: mapof.Any{"show-if": "attachmentOriginals.adapter eq S3"}},
					{Type: "text", Label: "Bucket", Path: "attachmentOriginals.bucket", Options: mapof.Any{"show-if": "attachmentOriginals.adapter eq S3"}},
					{Type: "text", Label: "Path", Path: "attachmentOriginals.path", Options: mapof.Any{"show-if": "attachmentOriginals.adapter eq S3"}},
				}},
				{Type: "layout-vertical", Label: "Cache", Children: []form.Element{
					{Type: "select", Label: "Adapter", Path: "attachmentCache.adapter"},
					{Type: "text", Label: "Location", Path: "attachmentCache.location"},
					{Type: "text", Label: "AccessKey", Path: "attachmentCache.accessKey", Options: mapof.Any{"show-if": "attachmentCache.adapter eq S3"}},
					{Type: "text", Label: "SecretKey", Path: "attachmentCache.secretKey", Options: mapof.Any{"show-if": "attachmentCache.adapter eq S3"}},
					{Type: "text", Label: "Token", Path: "attachmentCache.token", Options: mapof.Any{"show-if": "attachmentCache.adapter eq S3"}},
					{Type: "text", Label: "Region", Path: "attachmentCache.region", Options: mapof.Any{"show-if": "attachmentCache.adapter eq S3"}},
					{Type: "text", Label: "Bucket", Path: "attachmentCache.bucket", Options: mapof.Any{"show-if": "attachmentCache.adapter eq S3"}},
					{Type: "text", Label: "Path", Path: "attachmentCache.path", Options: mapof.Any{"show-if": "attachmentCache.adapter eq S3"}},
				}},
				{Type: "layout-vertical", Label: "Exports", Children: []form.Element{
					{Type: "select", Label: "Adapter", Path: "exportCache.adapter"},
					{Type: "text", Label: "Location", Path: "exportCache.location"},
					{Type: "text", Label: "AccessKey", Path: "exportCache.accessKey", Options: mapof.Any{"show-if": "exportCache.adapter eq S3"}},
					{Type: "text", Label: "SecretKey", Path: "exportCache.secretKey", Options: mapof.Any{"show-if": "exportCache.adapter eq S3"}},
					{Type: "text", Label: "Token", Path: "exportCache.token", Options: mapof.Any{"show-if": "exportCache.adapter eq S3"}},
					{Type: "text", Label: "Region", Path: "exportCache.region", Options: mapof.Any{"show-if": "exportCache.adapter eq S3"}},
					{Type: "text", Label: "Bucket", Path: "exportCache.bucket", Options: mapof.Any{"show-if": "exportCache.adapter eq S3"}},
					{Type: "text", Label: "Path", Path: "exportCache.path", Options: mapof.Any{"show-if": "exportCache.adapter eq S3"}},
				}},
			},
		}, false, nil

	case "certificates":
		return form.Element{
			Type:        "layout-vertical",
			Description: "Readable/Writeable location where SSL certificates are stored.",
			Children: []form.Element{
				{Type: "select", Label: "Adapter", Path: "certificates.adapter"},
				{Type: "text", Label: "Location", Path: "certificates.location"},
				{Type: "text", Label: "AccessKey", Path: "certificates.accessKey", Options: mapof.Any{"show-if": "certificates.adapter eq S3"}},
				{Type: "text", Label: "SecretKey", Path: "certificates.secretKey", Options: mapof.Any{"show-if": "certificates.adapter eq S3"}},
				{Type: "text", Label: "Bucket", Path: "certificates.bucket", Options: mapof.Any{"show-if": "certificates.adapter eq S3"}},
				{Type: "text", Label: "Path", Path: "certificates.path", Options: mapof.Any{"show-if": "certificates.adapter eq S3"}},
			},
		}, false, nil
	}

	return form.Element{}, false, derp.NewBadRequestError("handler.getSetupForm", "Invalid form name", name)
}
