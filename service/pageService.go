package service

import (
	"strings"

	"github.com/benpate/ghost/model"
	"github.com/labstack/echo/v4"
)

type PageService struct{}

func (pageService PageService) Render(ctx echo.Context, stream *model.Stream, view string) (string, string) {

	if ctx.Request().Header.Get("hx-request") == "true" {
		return pageService.RenderPartial(stream, view)
	}

	return pageService.RenderPage(stream, view)
}

func (pageService PageService) RenderPage(stream *model.Stream, view string) (string, string) {

	var header strings.Builder
	var footer strings.Builder

	innerHead, innerFoot := pageService.RenderPartial(stream, view)

	header.WriteString(`<html><head><title>GH0ST</title>`)
	header.WriteString(`<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/uikit/3.5.6/css/uikit.min.css">`)
	header.WriteString(`<link rel="stylesheet" href="/r/stylesheet.css">`)
	// header.WriteString(`<script src="https://unpkg.com/htmx.org@0.0.8"></script>`)
	header.WriteString(`</head><body hx-boost="true" hx-target="#stream" hx-push-url="true">`)
	header.WriteString(`<div>GLOBAL NAVIGATION HERE</di><hr>`)
	header.WriteString(`<div id="stream">`)
	header.WriteString(innerHead)

	footer.WriteString(innerFoot)
	footer.WriteString(`</div>`)
	footer.WriteString(`</body>`)
	footer.WriteString(`<script src="http://localhost/htmx/htmx.js"></script>`)
	footer.WriteString(`<script src="https://cdnjs.cloudflare.com/ajax/libs/uikit/3.5.6/js/uikit.min.js"></script>`)
	footer.WriteString(`</html>`)

	return header.String(), footer.String()
}

func (pageService PageService) RenderPartial(stream *model.Stream, view string) (string, string) {

	var header strings.Builder
	var footer strings.Builder

	header.WriteString(`<div hx-sse="/`)
	header.WriteString(stream.Token)
	header.WriteString(`/views/`)
	header.WriteString(view)
	header.WriteString(`/sse" hx-target="#stream-updates"></div>`)
	header.WriteString(`<div id="stream-updates">`)

	footer.WriteString(`</div>`)

	return header.String(), footer.String()
}
