package handler

import (
	"strings"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	domaintools "github.com/benpate/domain"
	"github.com/benpate/steranko"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

// GetQRCode generates a QR Code for the provided URL
func GetQRCode(ctx *steranko.Context, factory *domain.Factory, session data.Session) error {

	// Get the URL from the request; strip "/qrcode" from the path
	url := domaintools.Hostname(ctx.Request())
	url = domaintools.AddProtocol(url)
	url = url + strings.TrimSuffix(ctx.Request().URL.String(), "/qrcode")

	// Create a new QR code generator
	qrc, err := qrcode.New(url)

	if err != nil {
		return derp.Wrap(err, "build.StepQRCode.Get", "Error generating QR Code")
	}

	// Generate the QR code, and "save" it to the response writer
	w := standard.NewWithWriter(AsWriteCloser{ctx.Response().Writer})

	if err := qrc.Save(w); err != nil {
		return derp.Wrap(err, "build.StepQRCode.Get", "Error writing image")
	}

	// QR-on, my wayward son.
	return nil
}
