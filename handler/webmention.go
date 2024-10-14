package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/labstack/echo/v4"
)

func PostWebMention(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.PostWebMention"

	return func(ctx echo.Context) error {

		// This will receive form POST data from the webmention endpoint
		body := struct {
			Source string `form:"source"`
			Target string `form:"target"`
		}{}

		// Try to collect form data into the body struct
		if err := ctx.Bind(&body); err != nil {
			return derp.Wrap(err, location, "Invalid form data")
		}

		// Prepare a task to process the webmention asynchronously
		task := queue.NewTask("ReceiveWebMention", mapof.Any{
			"source": body.Source,
			"target": body.Target,
		})

		// Push the new task onto the background queue.
		if err := serverFactory.Queue().Publish(task); err != nil {
			return derp.Wrap(err, location, "Error queuing task", task)
		}

		// Success!  Return 201/Accepted to indicate that this request has been queued (which is true)
		return ctx.String(http.StatusAccepted, "Accepted")
	}
}
