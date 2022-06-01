package handler

import (
	"net/http"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/path"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/config"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
)

func GetServerIndex(factory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		domains := factory.ListDomains()

		b := html.New()

		pageHeader(ctx, b, "Domains")

		b.Script().Type("text/javascript").InnerHTML("function signOut() {document.cookie='admin=; Max-Age=-999999999;'; document.location.reload();}").Close()

		b.Div().ID("menu-bar").EndBracket()

		// Sign-Out
		b.Div().Class("right").EndBracket()
		b.A("").Script("on click call signOut()").EndBracket()
		b.Span().InnerHTML("Sign Out").Close()
		b.Space()
		b.I("fa-solid fa-arrow-right-from-bracket").Close()
		b.Close()
		b.Close()

		b.Close()

		b.H1().InnerHTML("Domains on this Server").Close()
		b.Div().Class("space-below").InnerHTML("Manage domains configured on this server.  For more settings, you can also edit the config.json file manually.").Close()

		// List existing domains
		b.Div().Class("pure-g").Data("hx-push-url", "false").EndBracket()

		// First row is "Add" link
		b.Div().Class("pure-u-1 pure-u-md-1-3 pure-u-lg-1-4 pure-u-xl-1-5")
		b.Div().Class("card").Role("link").Data("hx-get", factory.AdminURL()+"/new")
		b.Div().Class("align-center space-above space-below")
		{
			b.I("fa-4x fa-solid fa-plus-circle gray30").Close()
		}
		b.Close()
		b.H3().Class("align-center").InnerHTML("Add a Domain").Close()
		b.Div().Class("text-sm").InnerHTML("&nbsp;").Close()
		b.Close()
		b.Close()

		for index, d := range domains {
			indexString := convert.String(index)
			b.Div().Class("pure-u-1 pure-u-md-1-3 pure-u-lg-1-4 pure-u-xl-1-5")
			b.Div().Class("card")
			b.Div().Role("link").Data("hx-get", factory.AdminURL()+"/"+indexString)
			b.Div().Class("align-center space-above space-below")
			{
				b.I("fa-4x fa-solid fa-server gray30").Close()
			}
			b.Close()
			b.H3().Class("align-center").InnerHTML(d.Label)
			b.Close()

			b.Div().Class("text-sm align-center")

			// Show edit links
			if d.ConnectString == "" {
				b.A("").Data("hx-get", factory.AdminURL()+"/"+indexString).InnerHTML("CONFIGURE DOMAIN").Close()
			} else {
				b.A(factory.AdminURL()+"/"+indexString+"/signin").Attr("target", "_blank").InnerHTML("sign in").Close()
				b.Span().InnerHTML(" | ").Close()
				b.A("").Data("hx-get", factory.AdminURL()+"/"+indexString).InnerHTML("edit").Close()
			}

			// Server admin can delete all domains EXCEPT for localhost
			if d.Hostname != "localhost" {
				b.Span().InnerHTML(" | ").Close()
				b.Span().
					Class("red").
					Role("link").
					Data("hx-delete", factory.AdminURL()+"/"+indexString).
					Data("hx-confirm", "Delete this Domain?").
					InnerHTML("delete").
					Close()
			}

			b.Close()
			b.Close()
			b.Close()
		}

		// If there is a domain WITHOUT database info, then display its popup now.
		for index, domain := range domains {
			if domain.ConnectString == "" {
				indexString := convert.String(index)
				b.Div().Data("hx-get", factory.AdminURL()+"/"+indexString).Data("hx-trigger", "load").Close()
				break
			}
		}

		b.CloseAll()
		return ctx.HTML(http.StatusOK, b.String())
	}
}

func GetServerDomain(factory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		domain, err := factory.DomainByIndex(ctx.Param("domain"))

		if err != nil {
			return derp.Wrap(err, "handler.GetServerDomain", "Error loading configuration")
		}

		lib := factory.FormLibrary()

		f := form.Form{
			Kind:    "layout-tabs",
			Options: datatype.Map{"labels": "Server,Email"},
			Children: []form.Form{{
				Kind: "layout-vertical",
				Children: []form.Form{{
					Kind:        "text",
					Path:        "label",
					Label:       "Label",
					Description: "Admin-friendly label for this domain",
				}, {
					Kind:        "text",
					Path:        "hostname",
					Label:       "Hostname",
					Description: "Complete domain name (but no https:// or trailing slashes)",
				}, {
					Kind:        "text",
					Path:        "connectString",
					Label:       "MongoDB Connection String",
					Description: "Should look like mongodb://host:port/database",
				}, {
					Kind:        "text",
					Path:        "databaseName",
					Label:       "MongoDB Database Name",
					Description: "Name of the database to use on the server",
				}},
			}, {
				Kind: "layout-vertical",
				Children: []form.Form{{
					Kind:  "text",
					Path:  "smtp.hostname",
					Label: "SMTP Server",
				}, {
					Kind:  "text",
					Path:  "smtp.username",
					Label: "Username",
				}, {
					Kind:  "text",
					Path:  "smtp.password",
					Label: "Password",
				}, {
					Kind:  "checkbox",
					Path:  "smtp.tls",
					Label: "Use TLS Encryption",
				}},
			}},
		}

		s := config.Schema()
		formHTML, err := f.HTML(&lib, &s, &domain)

		if err != nil {
			return derp.Wrap(err, "handler.GetServerDomain", "Error generating form")
		}

		b := html.New()
		pageHeader(ctx, b, "Server Config")

		// Form Wrapper
		b.Form("post", ctx.Request().URL.String()).
			Attr("hx-post", ctx.Request().URL.String()).
			Attr("hx-swap", "#main").
			Attr("hx-push-url", "false").
			EndBracket()

		// Contents
		b.H1().InnerHTML("Domain Setup").Close()

		if domain.ConnectString == "" {
			b.Div().Class("space-below").InnerHTML("Welcome to server setup.  To begin, enter the database connection info for your local server.").Close()
		}
		b.WriteString(formHTML)

		// Controls
		b.Div()
		b.Button().Type("submit").Class("primary").TabIndex("0").InnerHTML("Save Changes").Close()
		b.Space()
		b.Span().Class("button").TabIndex("0").Script("on click trigger closeModal").InnerHTML("Cancel").Close()

		b.CloseAll()

		result := render.WrapModal(ctx.Response(), b.String())

		return ctx.HTML(200, result)
	}
}

func PostServerDomain(factory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		domainID := ctx.Param("domain")

		domain, err := factory.DomainByIndex(domainID)

		if err != nil {
			return derp.Wrap(err, "handler.PostServerDomain", "Error loading domain", ctx.Param("server"))
		}

		input := datatype.Map{}

		if err := (&echo.DefaultBinder{}).BindBody(ctx, &input); err != nil {
			return derp.Wrap(err, "handler.PostServerDomain", "Error binding form input")
		}

		s := config.Schema()

		if err := s.Validate(input); err != nil {
			return derp.Wrap(err, "handler.PostServerDomain", "Error validating input", domain)
		}

		if err := path.SetAll(&domain, input); err != nil {
			return derp.Wrap(err, "handler.PostServerDomain", "Error setting domain data", input)
		}

		if err := factory.UpdateDomain(domainID, domain); err != nil {
			return derp.Wrap(err, "handler.PostServerDomain", "Error saving domain")
		}

		render.CloseModal(ctx, "")
		return ctx.NoContent(http.StatusOK)
	}
}

// GetSigninToDomain signs you in to the requested domain as an administrator
func GetSigninToDomain(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.GetSigninToDomain"

	return func(ctx echo.Context) error {

		// Get the domain config requested in the URL (by index)
		domain, err := fm.DomainByIndex(ctx.Param("domain"))

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
		administrator.DisplayName = "System Administrator"
		administrator.IsOwner = true

		if err := s.CreateCertificate(ctx, &administrator); err != nil {
			return derp.Wrap(err, location, "Error creating certificate")
		}

		// Redirect to the admin page of this domain
		return ctx.Redirect(http.StatusTemporaryRedirect, "//"+domain.Hostname+"/startup")
	}
}
func DeleteServerDomain(factory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		domainID := ctx.Param("domain")

		domain, err := factory.DomainByIndex(domainID)

		if err != nil {
			return derp.Wrap(err, "handler.PostServerDomain", "Error loading domain", ctx.Param("server"))
		}

		if err := factory.DeleteDomain(domain); err != nil {
			return derp.Wrap(err, "handler.DeleteServerDomain", "Error deleting domain")
		}

		render.CloseModal(ctx, "")
		return ctx.NoContent(http.StatusOK)
	}
}

func pageHeader(ctx echo.Context, b *html.Builder, title string) {

	if ctx.Request().Header.Get("HX-Request") == "" {
		b.Container("html")
		b.Container("head")
		b.Container("title").InnerHTML(title).Close()

		b.Stylesheet("/static/purecss/pure-min.css")
		b.Stylesheet("/static/purecss/pure-grids-responsive-min.css")
		b.Stylesheet("/static/colors.css")

		b.Stylesheet("/static/accessibility.css")
		b.Stylesheet("/static/animations.css")
		b.Stylesheet("/static/cards.css")
		b.Stylesheet("/static/content.css")
		b.Stylesheet("/static/forms.css")
		b.Stylesheet("/static/layout.css")
		b.Stylesheet("/static/modal.css")
		b.Stylesheet("/static/responsive.css")
		b.Stylesheet("/static/tabs.css")
		b.Stylesheet("/static/tables.css")
		b.Stylesheet("/static/typography.css")
		b.Stylesheet("/static/fontawesome-free-6.0.0/css/all.css")

		b.Script().Src("/static/modal._hs").Type("text/hyperscript").Close()
		b.Script().Src("/static/forms._hs").Type("text/hyperscript").Close()
		b.Script().Src("/static/tabs._hs").Type("text/hyperscript").Close()
		b.Script().Src("/static/htmx/htmx.js").Close()
		b.Script().Src("/static/hyperscript/_hyperscript_web.min.js").Close()
		b.Script().Src("/static/a11y.js").Close()
		b.Script().Src("/static/extensions.js").Close()

		b.Close()
		b.Container("body")
		b.Container("aside").Close()
		b.Container("main")
		b.Div().ID("main").Class("framed")
		b.Div().ID("page").Data("hx-get", ctx.Request().URL.Path).Data("hx-trigger", "refreshPage from:window").Data("hx-target", "this").Data("hx-push-url", "false").EndBracket()
	}
}
