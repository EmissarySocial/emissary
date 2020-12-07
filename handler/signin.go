package handler

import (
	"bytes"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
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
			return derp.Report(derp.Wrap(err, "ghost.handler.GetSignin", "Error getting factory"))
		}

		steranko := factory.Steranko()

		return steranko.PostSignin(ctx)
	}
}

func PostSignOut(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return ctx.NoContent(200)
	}
}
