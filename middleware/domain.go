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
				builder.WriteString(`<html><head>`)
				builder.WriteString(`<script src="https://unpkg.com/htmx.org@0.0.8"></script>`)
				builder.WriteString(`</head><body>`)
				builder.WriteString(`<div>GLOBAL NAVIGATION HERE</di><hr>`)
				builder.WriteString(`<div id="stream" hx-target="#stream" hx-push-url="true">`)
				ctx.Response().Writer.Write([]byte(builder.String()))

				err := next(ctx)

				builder.Reset()
				builder.WriteString(`</div>`)
				builder.WriteString(`</body></html>`)
				ctx.Response().Writer.Write([]byte(builder.String()))

				return err
			}

			return next(ctx)
		}
	}
}
