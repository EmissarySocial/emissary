package handler

import (
	"strings"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

// StepQRCode represents an action-step that returns a QR Code for the current stream URL.
type StepQRCode struct {
	size int
}

// Get renders the Stream HTML to the context
func GetQRCode(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		var url string

		if ctx.Request().TLS == nil {
			url = "http://"
		} else {
			url = "https://"
		}

		url = url + ctx.Request().Host + strings.TrimSuffix(ctx.Request().URL.String(), "/qrcode")

		qrc, err := qrcode.New(url)

		if err != nil {
			return derp.Wrap(err, "render.StepQRCode.Get", "Error generating QR Code")
		}

		w := standard.NewWithWriter(AsWriteCloser{ctx.Response().Writer})

		// "save" file to the writer
		if err := qrc.Save(w); err != nil {
			return derp.Wrap(err, "render.StepQRCode.Get", "Error writing image")
		}

		return nil
	}
}
