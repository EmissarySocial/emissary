package handler

import (
	"strings"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	domaintools "github.com/benpate/domain"
	"github.com/labstack/echo/v4"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

// Get builds the Stream HTML to the context
func GetQRCode(fm *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		url := domaintools.Hostname(ctx.Request())
		url = domaintools.AddProtocol(url)
		url = url + strings.TrimSuffix(ctx.Request().URL.String(), "/qrcode")

		qrc, err := qrcode.New(url)

		if err != nil {
			return derp.Wrap(err, "build.StepQRCode.Get", "Error generating QR Code")
		}

		w := standard.NewWithWriter(AsWriteCloser{ctx.Response().Writer})

		// "save" file to the writer
		if err := qrc.Save(w); err != nil {
			return derp.Wrap(err, "build.StepQRCode.Get", "Error writing image")
		}

		return nil
	}
}
