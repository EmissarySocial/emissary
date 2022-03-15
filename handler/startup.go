package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/null"
	"github.com/benpate/schema"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

func GetStartupWelcome(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		b := html.New()
		pageHeader(ctx, b, "Let's Get Started")

		b.Div().Class("align-center")

		b.Div().Class("space-below")
		b.I("fa-8x fa-solid fa-volume-xmark gray20").Close()
		b.Close()

		b.H1().InnerHTML("Let's Set Up Your Whisperverse Server").Close()

		b.H2().Class("gray60", "space-below").InnerHTML("Three Quick Steps 'Till You're Up And Running").Close()

		b.Button().Class("primary").Data("hx-get", "/startup/username").Data("hx-push-url", "true").InnerHTML("Start Now &raquo;").Close()
		b.Close()

		return ctx.HTML(http.StatusOK, b.String())
	}
}

func GetStartupUsername(fm *server.Factory) echo.HandlerFunc {

	library := fm.FormLibrary()

	s := schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"sitename": schema.String{Format: "no-html"},
				"username": schema.String{Format: "no-html"},
				"password": schema.String{MinLength: null.NewInt(12)},
			},
		},
	}

	f := form.Form{
		Kind:  "layout-vertical",
		Label: "Step 1. Set Up Your Admin Account",
		Children: []form.Form{{
			Kind:        "text",
			Path:        "sitename",
			Label:       "Name of this Server",
			Description: "Choose a name for this server.  You can always change it later.",
		}, {
			Kind:        "text",
			Path:        "username",
			Label:       "Choose a Username",
			Description: "There are no other accounts yet, so you can use anything you want",
		}, {
			Kind:        "text",
			Path:        "password",
			Label:       "Choose a Password",
			Description: "This is important. Don't reuse passwords. Don't make it guessable.",
		}},
	}

	return func(ctx echo.Context) error {

		b := html.New()
		pageHeader(ctx, b, "Create Your Account")

		b.Form("post", "/startup/username").EndBracket()

		formHTML, err := f.HTML(&library, &s, nil)

		if err != nil {
			return derp.Wrap(err, "handler.GetStartupUsername", "Error generating username form")
		}

		b.WriteString(formHTML)
		b.Button().Type("submit").Class("primary").InnerHTML("Create My Account &raquo;").Close()

		return ctx.HTML(http.StatusOK, b.String())
	}
}

func PostStartupUsername(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return nil
	}
}

func GetStartupTopLevel(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return nil
	}
}

func PostStartupTopLevel(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return nil
	}
}
