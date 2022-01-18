package handler

import (
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

// GetRSS returns an RSS data feed for the requested URL
func GetRSS(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return err
		}

		service := factory.RSS()

		feed, err := service.Feed()

		if err != nil {
			err := derp.Wrap(err, "handler.GetRSS", "Error generating RSS feed")
			derp.Report(err)
			return err
		}

		// TODO: Replace these with real values from the server setup.
		feed.Title = "Title Goes Here"
		feed.Description = "Description Goes Here"
		feed.FeedUrl = "Feed URL goes here"

		result, errr := feed.ToJSON()

		if errr != nil {
			err := derp.Wrap(errr, "handler.GetRSS", "Error writing JSON feed information")
			derp.Report(err)
			return err
		}

		response := ctx.Response()
		response.Header().Set("content-type", "application/json")

		return ctx.String(200, result)
	}
}
