package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func DomainWrapper() echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			if ctx.Request().Header.Get("HX-Request") == "" {

				builder := &strings.Builder{}
				builder.WriteString(`<html><head><title>GH0ST</title>`)
				builder.WriteString(`<script src="http://localhost/htmx/htmx.js"></script>`)
				// builder.WriteString(`<script src="https://unpkg.com/htmx.org@0.0.8"></script>`)
				builder.WriteString(`</head><body>`)
				builder.WriteString(`<div>GLOBAL NAVIGATION HERE</di><hr>`)
				builder.WriteString(`<div hx-target="#stream" hx-push-url="true">`)
				// builder.WriteString(`<div hx-sse="connect /sse EventName">`)
				// builder.WriteString(`<div id="stream" hx-ws="connect ws://localhost/ws">`)
				// builder.WriteString(`<div id="stream">`)
				ctx.Response().Writer.Write([]byte(builder.String()))

				// TODO: real error handling here.
				err := next(ctx)

				builder.Reset()
				// builder.WriteString(`</div>`)
				builder.WriteString(`</div>`)
				builder.WriteString(`</div>`)
				builder.WriteString(`</body></html>`)
				ctx.Response().Writer.Write([]byte(builder.String()))

				return err
			}

			// Handle ETag here.
			if ctx.Request().Header.Get("If-None-Match") == "12345" {
				ctx.NoContent(304)
				return nil
			}

			ctx.Response().Header().Add("Etag", "12345")
			return next(ctx)
		}
	}
}
