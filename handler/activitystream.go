package handler

import (
	"fmt"

	"github.com/benpate/activitystream/reader"
	"github.com/benpate/activitystream/writer"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

// GetInbox returns an inbox for a particular ACTOR
func GetInbox(fm *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		w := writer.Block(
			writer.Person("John Connor", "en"),
			writer.Article().Summary("hey-oh", "en").Icon("http://www.blah.com"),
		)

		return ctx.JSON(200, w)
	}
}

// PostInbox accepts messages to a particular ACTOR
func PostInbox(fm *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var body []byte

		if _, err := ctx.Request().Body.Read(body); err != nil {
			return derp.Wrap(err, "whisper.PostInbox", "Error reading request body")
		}

		r, err := reader.New(body)

		if err != nil {
			return derp.Wrap(err, "whisper.PostInbox", "Error reading ActivityPub record")
		}

		fmt.Print(r)

		return nil
	}
}

// GetOutbox returns an inbox for a particular ACTOR
func GetOutbox(fm *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}

// PostOutbox accepts messages to a particular ACTOR
func PostOutbox(fm *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		return nil
	}
}
