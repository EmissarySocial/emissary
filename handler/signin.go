package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// GetSignIn generates an echo.HandlerFunc that handles GET /signin requests
func GetSignIn(factoryManager *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return executeDomainTemplate(factoryManager, ctx, "signin")
	}
}

// PostSignIn generates an echo.HandlerFunc that handles POST /signin requests
func PostSignIn(factoryManager *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.New(500, "handler.PostSignIn", "Invalid Request.  Please try again later.")
		}

		s := factory.Steranko()

		if err := s.SignIn(ctx); err != nil {
			ctx.Response().Header().Add("HX-Trigger", "SigninError")
			return ctx.HTML(derp.ErrorCode(err), derp.Message(err))
		}

		ctx.Response().Header().Add("HX-Trigger", "SigninSuccess")

		return ctx.NoContent(200)
	}
}

// PostSignOut generates an echo.HandlerFunc that handles POST /signout requests
func PostSignOut(factoryManager *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.New(500, "handler.PostSignOut", "Invalid Request.  Please try again later.")
		}

		s := factory.Steranko()

		if err := s.SignOut(ctx); err != nil {
			return derp.Wrap(err, "handler.PostSignOut", "Error Signing Out")
		}

		// Forward the user back to the home page of the website.
		ctx.Response().Header().Add("HX-Redirect", "/")
		return ctx.NoContent(http.StatusNoContent)
	}
}

func GetResetPassword(factoryManager *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return executeDomainTemplate(factoryManager, ctx, "reset-password")
	}
}

func PostResetPassword(factoryManager *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return executeDomainTemplate(factoryManager, ctx, "reset-password")
	}
}
