package handler

import (
	_ "embed"
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/labstack/echo/v4"
)

// smtpTestTimeout bounds the SMTP connectivity check run when a domain is saved, so an unreachable
// mail server fails in seconds instead of hanging the setup request.
const smtpTestTimeout = 10 * time.Second

// moderationTestTimeout bounds the moderation backend connectivity check run when a domain is saved.
const moderationTestTimeout = 10 * time.Second

// SetupDomainGet displays the form for creating/editing a domain.
func SetupDomainGet(factory *server.SetupFactory) echo.HandlerFunc {

	const location = "handler.SetupDomainGet"

	return func(ctx echo.Context) error {

		domainID := ctx.Param("domain")

		domain, err := factory.FindDomain(domainID)

		if err != nil {
			return derp.Wrap(err, location, "Loading configuration", domainID)
		}

		header := "Edit Domain"

		if domainID == "new" {
			header = "Add a Domain"
		}

		domainEditForm := setupDomainForm(header)

		s := schema.New(config.DomainSchema())

		formHTML, err := form.Editor(s, domainEditForm, &domain, nil)

		if err != nil {
			return derp.Wrap(err, location, "Generating form")
		}

		result := build.WrapModalForm(ctx.Response(), "/domains/"+domain.DomainID, formHTML, domainEditForm.Encoding())

		return ctx.HTML(200, result)
	}
}

// SetupDomainPost updates/creates a domain
func SetupDomainPost(serverFactory *server.SetupFactory) echo.HandlerFunc {

	const location = "handler.SetupDomainPost"

	return func(ctx echo.Context) error {

		domainID := ctx.Param("domain")

		// Try to load the existing domain.  If it does not exist, then create a new one.
		domain, _ := serverFactory.FindDomain(domainID)

		input := mapof.Any{}

		if err := (&echo.DefaultBinder{}).BindBody(ctx, &input); err != nil {
			return build.WrapInlineError(ctx.Response(), derp.Wrap(err, location, "Binding form input"))
		}

		// Update the domain configuration and save it to the domain storage (db/file/etc)
		s := schema.New(config.DomainSchema())

		if err := s.SetAll(&domain, input); err != nil {
			return build.WrapInlineError(ctx.Response(), derp.Wrap(err, location, "Setting config values"))
		}

		if _, err := s.Validate(&domain); err != nil {
			return build.WrapInlineError(ctx.Response(), derp.Wrap(err, location, "Validating config values"))
		}

		// Fail fast if the database is unreachable, so we don't hang on the driver's default
		// timeout or leave a broken domain persisted in the configuration file.  The error is
		// surfaced directly (not re-wrapped) so its specific, actionable message reaches the user.
		if err := serverFactory.TestConnection(domain); err != nil {
			return build.WrapInlineError(ctx.Response(), err)
		}

		// Verify the mail server is actually reachable (and the credentials work) before persisting,
		// so a typo'd host, wrong port, bad password, or mismatched TLS setting is caught HERE -- not
		// silently, the first time a member needs a password reset.  An unconfigured (empty) SMTP block
		// is allowed; TestConnection treats it as a no-op success.
		if err := domain.SMTPConnection.TestConnection(smtpTestTimeout); err != nil {
			return build.WrapInlineError(ctx.Response(), err)
		}

		// Verify the moderation backend is reachable (and the API key works) before persisting.
		// An unconfigured (empty) moderation block is allowed; TestConnection treats it as a no-op
		// success.  The URL is stored per-domain, not at the server level.
		if err := domain.Moderation.TestConnection(moderationTestTimeout); err != nil {
			return build.WrapInlineError(ctx.Response(), err)
		}

		// The error is surfaced directly (not re-wrapped) so actionable messages -- like
		// "configure the ActivityPub Cache first" -- reach the user.
		if err := serverFactory.PutDomain(domain); err != nil {
			return build.WrapInlineError(ctx.Response(), err)
		}

		build.CloseModal(ctx)
		build.RefreshPage(ctx)
		return ctx.NoContent(http.StatusOK)
	}
}

// SetupDomainDelete deletes a domain from the configuration
func SetupDomainDelete(factory *server.SetupFactory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		// Get the domain ID
		domainID := ctx.Param("domain")

		// Delete the domain
		if err := factory.DeleteDomain(domainID); err != nil {
			return derp.Wrap(err, "handler.SetupDomainDelete", "Deleting domain")
		}

		// Close the modal and return OK
		build.RefreshPage(ctx)
		return ctx.NoContent(http.StatusOK)
	}
}

// SetupDomainSigninPost signs you in to the requested domain as an administrator
func SetupDomainSigninPost(serverFactory *server.SetupFactory) echo.HandlerFunc {

	const location = "handler.SetupDomainSigninPost"

	return func(ctx echo.Context) error {

		// Get the domain config requested in the URL (by index)
		domain, err := serverFactory.FindDomain(ctx.Param("domain"))

		if err != nil {
			return derp.Wrap(err, location, "Loading configuration")
		}

		// Get the real factory for this domain
		factory, err := serverFactory.ByHostname(domain.Hostname)

		if err != nil {
			return derp.Wrap(err, location, "Loading Domain")
		}

		// Create a new database session
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return derp.Wrap(err, location, "Opening database session")
		}

		defer cancel()

		// Create a fake "User" record for the system administrator and sign in
		administrator := model.NewUser()
		administrator.DisplayName = "Server Administrator"
		administrator.IsOwner = true

		// Sign the Administrator into the system
		if err := factory.Steranko(session).SigninUser(ctx, &administrator); err != nil {
			return derp.Wrap(err, location, "Signing in administrator")
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
				Description: "Should look like: mongodb://127.0.0.1:27017/?directConnection=true&replicaSet=rs0",
			}, {
				Type:        "text",
				Path:        "databaseName",
				Label:       "MongoDB Database Name",
				Description: "Name of the database to use on the server",
			}},
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
			}},
		}, {
			Label: "Moderation",
			Type:  "layout-vertical",
			Children: []form.Element{{
				Type:        "select",
				Path:        "moderation.provider",
				Label:       "Provider",
				Description: "Select a moderation backend to forward reports to",
				Options: mapof.Any{"enum": []form.LookupCode{
					{Value: "", Label: "— None —"},
				}},
				Type:        "text",
				Path:        "moderation.url",
				Label:       "Backend URL",
				Description: "Base URL of the moderation backend (e.g. http://coop:3000)",
				Options:     mapof.Any{"show-if": "moderation.provider eq coop"},
			}},
		}},
	}
}
