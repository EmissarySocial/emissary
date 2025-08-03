package middleware

import (
	"fmt"
	"strings"

	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/re"
	"github.com/labstack/echo/v4"
)

func Debug() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {

			// Try to read the body from the request
			request := ctx.Request()
			body, err := re.ReadRequestBody(request)

			if err != nil {
				return derp.Wrap(err, "middleware.Debug", "Error reading body from request")
			}

			// Dump Request
			fmt.Println("")
			fmt.Println("-- Debugger Middleware -------------------")
			fmt.Println(request.Method + " " + request.URL.String() + " " + request.Proto)
			fmt.Println("Host: " + dt.Hostname(request))
			for key, value := range request.Header {
				fmt.Println(key + ": " + strings.Join(value, ", "))
			}
			fmt.Println("")
			fmt.Println(string(body))
			fmt.Println("")

			return next(ctx)
		}
	}
}
