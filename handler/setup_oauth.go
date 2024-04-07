package handler

import (
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/builder"
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
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
		result := builder.WrapModalForm(ctx.Response(), "/oauth/"+oAuthProviderID, editForm.Encoding(), formHTML)

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
		data := mapof.NewAny()

		if err := ctx.Bind(&data); err != nil {
			return derp.Wrap(err, "handler.SetupOAuthPost", "Error binding form data")
		}

		// Load the requested connection from the domain configuration.
		configuration := factory.Config()
		connection, _ := configuration.Providers.Get(providerID)
		connection.ProviderID = providerID

		// Apply the POST data into the connection
		editForm := setupOAuthForm("")
		if err := editForm.SetAll(&connection, data, nil); err != nil {
			return derp.Wrap(err, "handler.SetupOAuthPost", "Error applying form data")
		}

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
		builder.CloseModal(ctx)
		return ctx.NoContent(http.StatusOK)
	}
}

func setupOAuthForm(title string) form.Form {

	providerSchema := schema.New(config.ProviderSchema())

	return form.Form{
		Schema: providerSchema,
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       title,
			Description: "These credentials should be obtained from the provider's website when you register this OAuth client with their API.",
			Children: []form.Element{
				{
					Type:  "text",
					Label: "Client ID",
					Path:  "clientId",
					Options: mapof.Any{
						"autocomplete": "off",
					},
				},
				{
					Type:  "text",
					Label: "Client Secret",
					Path:  "clientSecret",
					Options: mapof.Any{
						"autocomplete": "off",
					},
				},
			},
		},
	}
}
