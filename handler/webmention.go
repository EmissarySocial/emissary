package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

func PostWebMention(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostWebMention"

	// This will receive form POST data from the webmention endpoint
	var body struct {
		Source string `form:"source"`
		Target string `form:"target"`
	}

	// Try to collect form data into the body struct
	if err := ctx.Bind(&body); err != nil {
		return derp.Wrap(err, location, "Invalid form data")
	}

	// Prepare a task to process the webmention asynchronously
	factory.Queue().NewTask("ReceiveWebMention", mapof.Any{
		"host":   factory.Hostname(),
		"source": body.Source,
		"target": body.Target,
	})

	// Success!  Return 201/Accepted to indicate that this request has been queued (which is true)
	return ctx.String(http.StatusAccepted, "Accepted")
}
