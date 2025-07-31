package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
)

func GetIntent_Continue(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User) error {
	url := first.String(ctx.QueryParam("url"), "/@me")
	return ctx.HTML(http.StatusOK, getIntent_Continue(url))
}

func getIntent_Continue(url string) string {

	// (close) directive can be handled without a confirmation page
	if url == "(close)" {
		return "<script>window.close();</script>"
	}

	// Otherwise, prevent open redirect attacks by
	// displaying a confirmation page that shows the next URL to the user
	b := html.New()

	b.HTML()
	b.Head()
	b.Link("stylesheet", "/.themes/global/stylesheet").Close()
	b.Link("stylesheet", "/.themes/default/stylesheet").Close()
	b.Close()

	b.Body()
	b.Div().ID("main").Style("display:none")
	b.H1().InnerText("Returning to Your Work").Close()
	b.Div().InnerText("Click here to return to your previous workflow").Close()
	b.Div().Class("bold").InnerText(url).Close()
	b.Div()
	b.A(url).InnerText("Continue &rarr;").Close()

	return b.String()
}
