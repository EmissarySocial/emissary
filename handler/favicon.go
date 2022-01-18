package handler

import (
	"os"

	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/server"
)

// GetFavicon returns the default favicon for this server
func GetFavicon(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		d := os.DirFS("./templates/static/favicon")
		f, err := d.Open("favicon.ico")

		if err != nil {
			return err
		}

		return ctx.Stream(200, "image/x-icon", f)
	}
}
