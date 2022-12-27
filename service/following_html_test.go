package service

import (
	"bytes"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/davecgh/go-spew/spew"
	"willnorris.com/go/microformats"
)

func TestMicroformats(t *testing.T) {

	var body bytes.Buffer
	txn := remote.Get("https://websub.rocks/blog/100/WD8utnz0FPRGbpvqO08R").Response(&body, nil)

	if err := txn.Send(); err != nil {
		derp.Report(err)
	}

	data := microformats.Parse(&body, txn.RequestObject.URL)

	spew.Dump(data)
}
