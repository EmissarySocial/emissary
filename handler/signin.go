package handler

import (
	"bytes"
	"net/http"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/server"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// GetSignIn generates an echo.HandlerFunc that handles GET /signin requests
func GetSignIn(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var buffer bytes.Buffer
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetSignin", "Error getting factory"))
		}

		template := factory.Layout().Template

		if err := template.ExecuteTemplate(&buffer, "signin", "error message goes here."); err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetSignin", "Error executing template"))
		}

		return ctx.HTML(200, buffer.String())
	}
}

// PostSignIn generates an echo.HandlerFunc that handles POST /signin requests
func PostSignIn(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			ctx.Response().Header().Add("HX-Trigger", "SigninError")
			return ctx.HTML(200, postSigninError("Invalid Request.  Please try again later."))
		}

		s := factory.Steranko()

		var txn steranko.SigninTransaction

		if err := ctx.Bind(&txn); err != nil {
			ctx.Response().Header().Add("HX-Trigger", "SigninError")
			return ctx.HTML(200, postSigninError("Invalid Request.  Please try again later."))
		}

		result := s.Signin(txn)

		if result.Error != nil {
			return ctx.HTML(200, postSigninError(derp.Message(result.Error)))
		}

		/*
			ctx.SetCookie(&http.Cookie{
				Name:   "Authentication",
				Value:  result.JWT,
				Secure: true,
			})
		*/

		// Success Response Headers
		ctx.Response().Header().Add("Authentication", result.JWT)
		ctx.Response().Header().Add("HX-Trigger", "SigninSuccess")

		return ctx.NoContent(200)
	}
}

func postSigninError(message string) string {
	return `<div class="uk-alert uk-alert-danger">` + message + `</div>`
}

// PostSignOut generates an echo.HandlerFunc that handles POST /signout requests
func PostSignOut(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		ctx.SetCookie(&http.Cookie{
			Name:    "Authentication",
			Value:   "",
			Secure:  true,
			Expires: time.Time{},
		})

		ctx.Redirect(http.StatusSeeOther, "/signin")

		return ctx.NoContent(200)
	}
}
