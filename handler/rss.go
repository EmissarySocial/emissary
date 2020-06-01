package handler

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// GetRSS returns an RSS data feed for the requested URL
func GetRSS(fm service.FactoryMaker) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory := fm.Factory(ctx.Request().Context())

		service := factory.RSS()

		feed, err := service.Feed()

		if err != nil {
			return derp.Wrap(err, "handler.GetRSS", "Error generating RSS feed").Report()
		}

		result, errr := feed.ToJSON()

		if errr != nil {
			return derp.New(500, "handler.GetRSS", "Error writing JSON feed information", errr).Report()
		}

		return ctx.String(200, result)
	}
}
