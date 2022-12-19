package handler

import (
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
	"github.com/labstack/echo/v4"
)

func SetupOAuthList(factory *server.Factory, templates *template.Template) echo.HandlerFunc {
	return SetupPageGet(factory, templates, "oauth.html")
}

func SetupOAuthGet(factory *server.Factory, templates *template.Template) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Locate the requested provider
		oAuthProviders := set.Slice[form.LookupCode](dataset.Providers())
		oAuthProviderID := ctx.Param("provider")
		oAuthProvider, ok := oAuthProviders.Get(oAuthProviderID)

		if !ok {
			return derp.NewInternalError("setup.oauth.get", "Unable to find provider", oAuthProviderID)
		}

		// Build the custom form title
		iconService := factory.Icons()
		title := iconService.Get(oAuthProvider.Icon) + " " + oAuthProvider.Label + " - OAuth Setup"

		// Load the requested connection from the domain configuration.
		configuration := factory.Config()
		connection, _ := configuration.Providers.Get(oAuthProviderID)
		connection.ProviderID = oAuthProviderID

		// Create the form to edit the data
		editForm := setupOAuthForm(title)
		formHTML, err := editForm.Editor(connection, nil)

		if err != nil {
			return derp.Wrap(err, "handler.SetupDomainGet", "Error generating form")
		}

		// Wrap the form in a modal dialog
		result := render.WrapModalForm(ctx.Response(), "/oauth/"+oAuthProviderID, formHTML)

		return ctx.HTML(200, result)
	}
}

func SetupOAuthPost(factory *server.Factory, templates *template.Template) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Validate provider ID
		providers := set.Slice[form.LookupCode](dataset.Providers())
		providerID := ctx.Param("provider")

		if _, ok := providers.Get(providerID); !ok {
			return derp.NewInternalError("setup.oauth.get", "Unable to find provider", providerID)
		}

		// Collect the updated connection information from the form post
		data := maps.New()

		if err := ctx.Bind(&data); err != nil {
			return derp.Wrap(err, "handler.SetupOAuthPost", "Error binding form data")
		}

		// Load the requested connection from the domain configuration.
		configuration := factory.Config()
		connection, _ := configuration.Providers.Get(providerID)
		connection.ProviderID = providerID

		// Apply the POST data into the connection
		editForm := setupOAuthForm("")
		editForm.SetAll(&connection, data, nil)

		// Save/Delete the connection
		if !connection.IsEmpty() {

			// Save to configuration location
			if err := factory.PutProvider(connection); err != nil {
				return derp.Wrap(err, "handler.SetupOAuthPost", "Error saving connection", connection)
			}

		} else {

			// Remove empty configurations
			if err := factory.DeleteProvider(providerID); err != nil {
				return derp.Wrap(err, "handler.SetupOAuthPost", "Error saving connection", providerID)
			}
		}

		// Success!
		render.CloseModal(ctx, "")
		return ctx.NoContent(http.StatusOK)
	}
}

func setupOAuthForm(title string) form.Form {

	return form.Form{
		Schema: config.ProviderSchema(),
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       title,
			Description: "These credentials should be obtained from the provider's website when you register this OAuth client with their API.",
			Children: []form.Element{
				{
					Type:  "text",
					Label: "Client ID",
					Path:  "clientId",
					Options: maps.Map{
						"autocomplete": "off",
					},
				},
				{
					Type:  "text",
					Label: "Client Secret",
					Path:  "clientSecret",
					Options: maps.Map{
						"autocomplete": "off",
					},
				},
			},
		},
	}
}
