package handler

import (
	_ "embed"
	"net/http"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/labstack/echo/v4"
)

// SetupDomainGet displays the form for creating/editing a domain.
func SetupDomainGet(factory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		var header string

		domainID := ctx.Param("domain")

		if domainID == "new" {
			header = "Add a Domain"
		} else {
			header = "Edit a Domain"
		}

		domain, err := factory.DomainByID(domainID)

		if err != nil {
			return derp.Wrap(err, "handler.SetupDomainGet", "Error loading configuration")
		}

		domainEditForm := setupDomainForm(header)

		s := schema.New(config.DomainSchema())

		formHTML, err := form.Editor(s, domainEditForm, domain, nil)

		if err != nil {
			return derp.Wrap(err, "handler.SetupDomainGet", "Error generating form")
		}

		result := render.WrapModalForm(ctx.Response(), "/domains/"+domain.DomainID, formHTML)

		return ctx.HTML(200, result)
	}
}

// SetupDomainPost updates/creates a domain
func SetupDomainPost(factory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		domainID := ctx.Param("domain")

		// Try to load the existing domain.  If it does not exist, then create a new one.
		domain, _ := factory.DomainByID(domainID)

		input := mapof.Any{}

		if err := (&echo.DefaultBinder{}).BindBody(ctx, &input); err != nil {
			return render.WrapInlineError(ctx, derp.Wrap(err, "handler.SetupDomainPost", "Error binding form input"))
		}

		s := schema.New(config.DomainSchema())

		if err := s.SetAll(&domain, input); err != nil {
			return render.WrapInlineError(ctx, derp.Wrap(err, "handler.SetupDomainPost", "Error setting config values"))
		}

		if err := s.Validate(&domain); err != nil {
			return render.WrapInlineError(ctx, derp.Wrap(err, "handler.SetupDomainPost", "Error validating config values"))
		}

		if err := factory.PutDomain(domain); err != nil {
			return render.WrapInlineError(ctx, derp.Wrap(err, "handler.SetupDomainPost", "Error saving domain"))
		}

		render.CloseModal(ctx, "")
		return ctx.NoContent(http.StatusOK)
	}
}

// SetupDomainDelete deletes a domain from the configuration
func SetupDomainDelete(factory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		// Get the domain ID
		domainID := ctx.Param("domain")

		// Delete the domain
		if err := factory.DeleteDomain(domainID); err != nil {
			return derp.Wrap(err, "handler.SetupDomainDelete", "Error deleting domain")
		}

		// Close the modal and return OK
		render.RefreshPage(ctx)
		return ctx.NoContent(http.StatusOK)
	}
}

// SetupDomainSigninPost signs you in to the requested domain as an administrator
func SetupDomainSigninPost(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.SetupDomainSigninPost"

	return func(ctx echo.Context) error {

		// Get the domain config requested in the URL (by index)
		domain, err := fm.DomainByID(ctx.Param("domain"))

		if err != nil {
			return derp.Wrap(err, location, "Error loading configuration")
		}

		// Get the real factory for this domain
		factory, err := fm.ByDomainName(domain.Hostname)

		if err != nil {
			return derp.Wrap(err, location, "Error loading Domain")
		}

		// Create a fake "User" record for the system administrator and sign in
		s := factory.Steranko()

		administrator := model.NewUser()
		administrator.DisplayName = "Server Administrator"
		administrator.IsOwner = true

		if err := s.CreateCertificate(ctx, &administrator); err != nil {
			return derp.Wrap(err, location, "Error creating certificate")
		}

		// Redirect to the admin page of this domain
		return ctx.Redirect(http.StatusTemporaryRedirect, "//"+domain.Hostname+"/startup")
	}
}

func setupDomainForm(header string) form.Element {
	return form.Element{
		Type:  "layout-tabs",
		Label: header,
		Children: []form.Element{{
			Label: "Domain",
			Type:  "layout-vertical",
			Children: []form.Element{{
				Type:        "text",
				Path:        "label",
				Label:       "Label",
				Description: "Admin-friendly label for this domain",
			}, {
				Type:        "text",
				Path:        "hostname",
				Label:       "Hostname",
				Description: "Complete domain name (but no https:// or trailing slashes)",
			}, {
				Type:        "text",
				Path:        "connectString",
				Label:       "MongoDB Connection String",
				Description: "Should look like mongodb://host:port/database",
			}, {
				Type:        "text",
				Path:        "databaseName",
				Label:       "MongoDB Database Name",
				Description: "Name of the database to use on the server",
			}},
		}, {
			Label: "SMTP Setup",
			Type:  "layout-vertical",
			Children: []form.Element{{
				Type:  "text",
				Path:  "smtp.hostname",
				Label: "Hostname",
			}, {
				Type:  "text",
				Path:  "smtp.username",
				Label: "Username",
			}, {
				Type:  "text",
				Path:  "smtp.password",
				Label: "Password",
			}, {
				Type:  "text",
				Path:  "smtp.port",
				Label: "Port",
			}, {
				Type:  "toggle",
				Path:  "smtp.tls",
				Label: "Use TLS?",
			},
			},
		}, {
			Label: "Account Owner",
			Type:  "layout-vertical",
			Children: []form.Element{
				{
					Type:  "text",
					Path:  "owner.displayName",
					Label: "Name",
				},
				{
					Type:        "text",
					Path:        "owner.username",
					Label:       "Username",
					Description: "The username for this account",
				},
				{
					Type:        "text",
					Path:        "owner.emailAddress",
					Label:       "Email Address",
					Description: "A welcome email will be sent to this address",
				},
				{
					Type:  "text",
					Path:  "owner.phoneNumber",
					Label: "Phone Number",
				},
				{
					Type:  "textarea",
					Path:  "owner.mailingAddress",
					Label: "Mailing Address",
					Options: mapof.Any{
						"rows": "3",
					},
				},
			},
		}},
	}
}
