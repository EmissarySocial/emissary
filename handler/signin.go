package handler

import (
	"bytes"
	"net/http"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
)

func GetSignIn(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var buffer bytes.Buffer
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetSignin", "Error getting factory"))
		}

		layout := factory.Layout()

		template := layout.Layout()

		if err := template.ExecuteTemplate(&buffer, "signin", "error message goes here."); err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetSignin", "Error executing template"))
		}

		return ctx.HTML(200, buffer.String())
	}
}

func PostSignIn(factoryManager *service.FactoryManager) echo.HandlerFunc {

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

		spew.Dump(txn)
		result := s.Signin(txn)
		spew.Dump(result)

		if result.Error != nil {
			return ctx.HTML(200, postSigninError(derp.Message(result.Error)))
		}

		ctx.SetCookie(&http.Cookie{
			Name:   "Authentication",
			Value:  result.JWT,
			Secure: true,
		})

		// Success Response Headers
		// ctx.Response().Header().Add("Authentication", result.JWT)
		ctx.Response().Header().Add("HX-Trigger", "SigninSuccess")

		return ctx.NoContent(200)
	}
}

func postSigninError(message string) string {
	return `<div class="uk-alert uk-alert-danger">` + message + `</div>`
}

func PostSignOut(factoryManager *service.FactoryManager) echo.HandlerFunc {

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
