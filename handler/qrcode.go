package handler

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

func GetQRCode_Stream(ctx *steranko.Context, factory *service.Factory, _ data.Session, stream *model.Stream) error {
	return getQRCode(ctx, stream.ActivityPubURL())
}

func GetQRCode_User(ctx *steranko.Context, factory *service.Factory, _ data.Session, user *model.User) error {
	return getQRCode(ctx, user.ActivityPubURL())
}

// getQRCode generates a QR Code for the provided URL
func getQRCode(ctx *steranko.Context, url string) error {

	// Pass query strings through to the QR code.
	if rawQuery := ctx.Request().URL.RawQuery; rawQuery != "" {
		url += "?" + rawQuery
	}

	spew.Dump(url)

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
