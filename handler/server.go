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
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
)

func GetServerIndex(factoryManager *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		domains := factoryManager.ListDomains()

		b := html.New()

		pageHeader(ctx, b, "Domains")

		b.H1().InnerHTML("Server Admin").Close()
		b.Table().Class("table").EndBracket()

		// Add a new record
		b.TR().Data("hx-get", "/server/new")
		b.TD().Class("link").Attr("colspan", "2")
		b.I("fa-solid fa-plus-circle").Close()
		b.Space()
		b.Span().InnerHTML("Add a Domain").Close()
		b.Close()
		b.Close()

		// Display existing records
		for index, d := range domains {
			indexString := convert.String(index)
			b.TR()
			b.TD().Attr("nowrap", "true").Data("hx-get", "/server/"+indexString)
			b.I("fa-solid fa-server").Close()
			b.Space()
			b.Span().InnerHTML(d.Label).Close()
			b.Close()
			b.TD().Data("hx-get", "/server/"+indexString).Style("width:100%;").InnerHTML(d.Hostname).Close()
			b.TD()
			b.I("fa-solid fa-trash").
				Data("hx-delete", "/server/"+indexString).
				Data("hx-confirm", "Delete this Domain?").
				Close()
			b.Close()
		}

		b.CloseAll()
		return ctx.HTML(http.StatusOK, b.String())
	}
}

func GetServerDomain(factoryManager *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		domain, err := factoryManager.DomainByIndex(ctx.Param("domain"))

		if err != nil {
			return derp.Wrap(err, "whisper.handler.GetServerDomain", "Error loading Domain config")
		}

		lib := factoryManager.FormLibrary()

		f := form.Form{
			Kind:    "layout-tabs",
			Options: form.Map{"labels": "Server,Email"},
			Children: []form.Form{{
				Kind: "layout-vertical",
				Children: []form.Form{{
					Kind:        "text",
					Path:        "label",
					Label:       "Label",
					Description: "Admin-friendly label for this domain",
				}, {
					Kind:  "text",
					Path:  "hostname",
					Label: "Hostname",
				}, {
					Kind:  "text",
					Path:  "connectString",
					Label: "MongoDB Connection String",
				}, {
					Kind:  "text",
					Path:  "databaseName",
					Label: "MongoDB Database Name",
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
			return derp.Wrap(err, "whisper.handler.GetServerDomain", "Error generating form")
		}

		b := html.New()
		pageHeader(ctx, b, "Server Config")

		// Form Wrapper
		b.Form("post", "").
			Attr("hx-post", ctx.Request().URL.String()).
			Attr("hx-swap", "#main").
			Attr("hx-push-url", "false").
			EndBracket()

		// Contents
		b.H1().InnerHTML("Domain Settings").Close()
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

func PostServerDomain(factoryManager *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		domainID := ctx.Param("domain")

		domain, err := factoryManager.DomainByIndex(domainID)

		if err != nil {
			return derp.Wrap(err, "whisper.handler.PostServerDomain", "Error loading domain", ctx.Param("server"))
		}

		input := datatype.Map{}

		if err := (&echo.DefaultBinder{}).BindBody(ctx, &input); err != nil {
			return derp.Wrap(err, "whisper.handler.PostServerDomain", "Error binding form input")
		}

		s := config.Schema()

		if err := s.Validate(input); err != nil {
			return derp.Wrap(err, "whisper.handler.PostServerDomain", "Error validating input", domain)
		}

		if err := path.SetAll(&domain, input); err != nil {
			return derp.Wrap(err, "whisper.handler.PostServerDomain", "Error setting domain data", input)
		}

		if err := factoryManager.UpdateDomain(domainID, domain); err != nil {
			return derp.Wrap(err, "whisper.handler.PostServerDomain", "Error saving domain")
		}

		render.CloseModal(ctx, "")
		return ctx.NoContent(http.StatusOK)
	}
}

func DeleteServerDomain(factoryManager *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		domainID := ctx.Param("domain")

		domain, err := factoryManager.DomainByIndex(domainID)

		if err != nil {
			return derp.Wrap(err, "whisper.handler.PostServerDomain", "Error loading domain", ctx.Param("server"))
		}

		if err := factoryManager.DeleteDomain(domain); err != nil {
			return derp.Wrap(err, "whisper.handler.DeleteServerDomain", "Error deleting domain")
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

		b.Link("stylesheet", "/static/pure-min.css")
		b.Link("stylesheet", "/static/pure-grids-responsive-min.css")
		b.Link("stylesheet", "/static/colors.css")

		b.Link("stylesheet", "/static/accessibility.css")
		b.Link("stylesheet", "/static/animations.css")
		b.Link("stylesheet", "/static/cards.css")
		b.Link("stylesheet", "/static/content.css")
		b.Link("stylesheet", "/static/forms.css")
		b.Link("stylesheet", "/static/layout.css")
		b.Link("stylesheet", "/static/modal.css")
		b.Link("stylesheet", "/static/responsive.css")
		b.Link("stylesheet", "/static/tabs.css")
		b.Link("stylesheet", "/static/tables.css")
		b.Link("stylesheet", "/static/typography.css")
		b.Link("stylesheet", "/static/fontawesome-free-6.0.0/css/all.css")

		b.Container("script").Attr("src", "/htmx/htmx.js").Close()
		b.Container("script").Attr("src", "/static/modal.hs").Attr("type", "text/hyperscript").Close()
		b.Container("script").Attr("src", "/static/tabs.hs").Attr("type", "text/hyperscript").Close()
		b.Container("script").Attr("src", "https://unpkg.com/hyperscript.org").Close()

		b.Close()
		b.Container("body")
		b.Container("aside").Close()
		b.Container("main")
		b.Div().ID("main").Class("framed")
		b.Div().ID("page").Data("hx-get", ctx.Request().URL.Path).Data("hx-trigger", "refreshPage from:window").Data("hx-target", "this").EndBracket()
	}
}
