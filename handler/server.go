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

		b.Stylesheet("/.themes/global/stylesheet").Close()
		b.Script().Src("/.themes/global/hyperscript").Type("text/hyperscript").Close()

		b.Close()
		b.Container("body")
		b.Container("aside").Close()
		b.Container("main")
		b.Div().ID("main").Class("framed")
		b.Div().ID("page").Data("hx-get", ctx.Request().URL.Path).Data("hx-trigger", "refreshPage from:window").Data("hx-target", "this").Data("hx-push-url", "false").EndBracket()
	}
}
