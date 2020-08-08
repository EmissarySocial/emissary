package handler

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

func GetTransition(factoryMaker service.FactoryMaker) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		token := ctx.Param("token")
		transitionID := ctx.Param("transitionId")

		factory := factoryMaker.Factory(ctx.Request().Context())

		streamService := factory.Stream()
		templateService := factory.Template()
		library := factory.FormLibrary()

		stream, err := streamService.LoadByToken(token)

		if err != nil {
			err = derp.Wrap(err, "ghost.handler.GetTransition", "Cannot load Stream", token)
			derp.Report(err)
			return err
		}

		template, err := templateService.Load(stream.Template)

		transition := template.Transition(transitionID)

		// TODO: Validate that this transition is VALID
		// TODO: Validate that the USER IS PERMITTED to make this transition.

		if transition == nil {
			err = derp.New(404, "ghost.handler.GetTransition", "Unrecognized Transition", transitionID)
		}

		// Generate HTML by merging the form with the element library, the data schema, and the data value
		html, errr := transition.Form.HTML(library, template.Schema, stream.Data)

		if errr != nil {
			return derp.Report(derp.Wrap(errr, "ghost.handler.getTransition", "Error generating form HTML", transition.Form))
		}

		// Success!
		return ctx.HTML(200, html)
	}
}

func PostTransition(factoryMaker service.FactoryMaker) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return nil
	}
}
