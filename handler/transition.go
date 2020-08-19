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
		pageService := factory.PageService()
		formService := factory.FormService()
		library := factory.FormLibrary()

		stream, err := streamService.LoadByToken(token)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetTransition", "Cannot load Stream", token))
		}

		template, err := templateService.Load(stream.Template)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetTransition", "Cannot load Template", stream.Template))
		}

		transition, err := template.Transition(stream.State, transitionID)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetTransition", "Invalid Transition"))
		}

		// TODO: Validate that this transition is VALID
		// TODO: Validate that the USER IS PERMITTED to make this transition.

		if transition == nil {
			err = derp.New(404, "ghost.handler.GetTransition", "Unrecognized Transition", transitionID)
		}

		form, err := template.Form(stream.State, transitionID)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.getTransition", "Invalid Form"))
		}

		// Generate HTML by merging the form with the element library, the data schema, and the data value
		html, errr := form.HTML(library, *template.Schema, stream)

		if errr != nil {
			return derp.Report(derp.Wrap(errr, "ghost.handler.getTransition", "Error generating form HTML", form))
		}

		header, footer := pageService.Render(ctx, stream, "")

		formTop, formBottom := formService.Render(stream, transition)

		// Success!
		response := ctx.Response()
		response.WriteHeader(200)
		response.Write([]byte(header))
		response.Write([]byte(formTop))
		response.Write([]byte(html))
		response.Write([]byte(formBottom))
		response.Write([]byte(footer))

		return nil
	}
}

func PostTransition(factoryMaker service.FactoryMaker) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Get parameters from context
		token := ctx.Param("token")
		transitionID := ctx.Param("transitionId")

		form := make(map[string]interface{})

		if err := ctx.Bind(&form); err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Cannot load parse form data"))
		}

		// Get Factory and services required for this step
		factory := factoryMaker.Factory(ctx.Request().Context())
		streamService := factory.Stream()
		templateService := factory.Template()

		nextView := "default"

		// Load stream
		stream, err := streamService.LoadByToken(token)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Cannot load stream", token))
		}

		// Load template
		template, err := templateService.Load(stream.Template)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Cannot load template", stream))
		}

		// Execute transition
		if transition, err := template.Transition(stream.State, transitionID); err == nil {

			if err := streamService.Transition(stream, template, transitionID, form); err != nil {
				return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Error updating stream"))
			}

			nextView = transition.NextView
		}

		// Render the "next" view
		pageService := factory.PageService()

		// Generate the result
		result, err := streamService.Render(stream, nextView)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostTransition", "Error rendering stream"))
		}

		header, footer := pageService.Render(ctx, stream, nextView)

		// Success!
		response := ctx.Response()
		response.WriteHeader(200)
		response.Write([]byte(header))
		response.Write([]byte(result))
		response.Write([]byte(footer))

		return nil
	}
}
