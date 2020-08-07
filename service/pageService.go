package service

import (
	"strings"

	"github.com/benpate/ghost/model"
)

type PageService struct{}

func (pageService PageService) RenderPage(stream *model.Stream, view string) (string, string) {

	var header strings.Builder
	var footer strings.Builder

	header.WriteString(`<html><head><title>GH0ST</title>`)
	header.WriteString(`<script src="http://localhost/htmx/htmx.js"></script>`)
	// header.WriteString(`<script src="https://unpkg.com/htmx.org@0.0.8"></script>`)
	header.WriteString(`</head><body>`)
	header.WriteString(`<div>GLOBAL NAVIGATION HERE</di><hr>`)
	header.WriteString(`<div hx-target="#stream" hx-push-url="true">`)

	header.WriteString(`<div id="stream" hx-sse="connect /`)
	header.WriteString(stream.StreamID.Hex())
	header.WriteString(`/`)
	header.WriteString(view)
	header.WriteString(`/sse EventName">`)
	header.WriteString(`<div id="stream-updates">`)

	footer.WriteString(`</div>`)
	footer.WriteString(`</div>`)
	footer.WriteString(`</div>`)
	footer.WriteString(`</body></html>`)

	return header.String(), footer.String()

}

func (pageService PageService) RenderPartial(stream *model.Stream, view string) (string, string) {

	var header strings.Builder
	var footer strings.Builder

	header.WriteString(`<div id="stream" hx-sse="connect /`)
	header.WriteString(stream.StreamID.Hex())
	header.WriteString(`/`)
	header.WriteString(view)
	header.WriteString(`/sse EventName">`)
	header.WriteString(`<div id="stream-updates">`)

	footer.WriteString(`</div>`)
	footer.WriteString(`</div>`)

	return header.String(), footer.String()
}
