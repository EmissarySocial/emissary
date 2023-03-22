package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

func PostWebMention(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.PostWebMention"

	return func(ctx echo.Context) error {

		// Try to collect the form data
		body := struct {
			Source string `form:"source"`
			Target string `form:"target"`
		}{}

		if err := ctx.Bind(&body); err != nil {
			return derp.Wrap(err, location, "Invalid form data")
		}

		// Try to locate the requested domain
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Run the rest of the process asynchronously
		factory.Queue().Run(
			service.NewTaskReceiveWebMention(
				factory.Stream(),
				factory.Mention(),
				factory.User(),
				body.Source,
				body.Target,
			),
		)

		// Success!  Return 201/Accepted to indicate that this request has been queued (which is true)
		return ctx.String(http.StatusAccepted, "Accepted")
	}
}
