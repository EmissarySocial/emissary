package handler

import (
	"github.com/benpate/html"
	"github.com/labstack/echo/v4"
)

func pageHeader(ctx echo.Context, b *html.Builder, title string) {

	if ctx.Request().Header.Get("HX-Request") == "" {
		b.Container("html")
		b.Container("head")
		b.Container("title").InnerText(title).Close()

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
		b.Script().Src("/static/htmx/htmx.min.js").Close()
		b.Script().Src("/static/hyperscript/_hyperscript.min.js").Close()
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
