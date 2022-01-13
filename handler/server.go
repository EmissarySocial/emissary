package handler

import (
	"net/http"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/config"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/server"
	"github.com/benpate/html"
	"github.com/benpate/path"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
)

func GetServerIndex(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		domains := factoryManager.Domains()

		b := html.New()

		pageHeader(ctx, b, "Domain List")

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
			b.TR().Data("hx-get", "/server/"+convert.String(index))
			b.TD()
			b.I("fa-solid fa-server").Close()
			b.Space()
			b.Span().InnerHTML(d.Label).Close()
			b.Close()
			b.TD().InnerHTML(d.Hostname).Close()
			b.Close()
		}

		b.CloseAll()
		return ctx.HTML(http.StatusOK, b.String())
	}
}

func GetServerDomain(factoryManager *server.FactoryManager) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		domain, err := factoryManager.DomainByIndex(ctx.Param("server"))

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetServer", "Error loading Domain config")
		}

		lib := factoryManager.FormLibrary()

		f := form.Form{
			Kind:    "layout-vertical",
			Options: form.Map{"show-labels": "false"},
			Children: []form.Form{{
				Kind:  "layout-vertical",
				Label: "Server Details",
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
				Kind:  "layout-vertical",
				Label: "Email Server",
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
			return derp.Wrap(err, "ghost.handler.GetServer", "Error generating form")
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

func PostServerDomain(factoryManager *server.FactoryManager) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		domainID := ctx.Param("server")

		domain, err := factoryManager.DomainByIndex(domainID)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.PostServer", "Error loading domain", ctx.Param("server"))
		}

		input := datatype.Map{}

		if err := (&echo.DefaultBinder{}).BindBody(ctx, &input); err != nil {
			return derp.Wrap(err, "ghost.handler.PostServer", "Error binding form input")
		}

		spew.Dump(input)
		s := config.Schema()

		if err := s.Validate(input); err != nil {
			return derp.Wrap(err, "ghost.handler.PostServer", "Error validating input", domain)
		}
		spew.Dump(input)

		if err := path.SetAll(&domain, input); err != nil {
			return derp.Wrap(err, "ghost.handler.PostServer", "Error setting domain data", input)
		}

		if err := factoryManager.UpdateDomain(domainID, domain); err != nil {
			return derp.Wrap(err, "ghost.handler.PostServer", "Error saving domain")
		}

		if err := factoryManager.WriteConfig(); err != nil {
			return derp.Wrap(err, "ghost.handler.PostServer", "Error writing configuration file")
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
		b.Container("script").Attr("src", "https://unpkg.com/hyperscript.org").Close()

		b.Close()
		b.Container("body")
		b.Container("aside").Close()
		b.Container("main")
		b.Div().ID("main").Class("framed")
		b.Div().ID("page").Data("hx-get", ctx.Request().URL.Path).Data("hx-trigger", "refreshPage from:window").Data("hx-target", "this").EndBracket()
	}
}
