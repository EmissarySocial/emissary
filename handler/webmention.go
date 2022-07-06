package handler

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

func PostWebMention(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.PostWebMention"

	return func(ctx echo.Context) error {

		// Try to locate the requested domain
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Try to bind the form data
		body := struct {
			Source string `form:"source"`
			Target string `form:"target"`
		}{}

		if err := ctx.Bind(&body); err != nil {
			return derp.Wrap(err, location, "Invalid form data")
		}

		// Try to link the mention to a local page
		targetURL, err := url.Parse(body.Target)

		if err != nil {
			return derp.Wrap(err, location, "Cannot parse target URL", body.Target)
		}

		if targetURL.Hostname() != factory.Hostname() {
			return derp.Wrap(err, location, "Invalid Target URL")
		}

		// TODO: This action should be queued via a channel (len=10?).

		streamService := factory.Stream()
		stream := model.NewStream()
		token := strings.TrimPrefix(targetURL.Path, "/")

		if err := streamService.LoadByToken(token, &stream); err != nil {
			return derp.Wrap(err, location, "Cannot load stream", token)
		}

		// Try to validate the WebMention data
		mentionService := factory.Mention()

		if err := mentionService.Verify(body.Source, body.Target); err != nil {
			return derp.Wrap(err, location, "Source does not link to target", body.Source, body.Target)
		}

		// Write the mention to the database.
		mention := model.NewMention()
		mention.StreamID = stream.StreamID
		mention.Source = body.Source

		if err := mentionService.Save(&mention, "Created"); err != nil {
			return derp.Wrap(err, location, "Error saving mention")
		}

		return nil
	}
}
