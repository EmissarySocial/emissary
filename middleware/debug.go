package middleware

/*
THIS MIDDLEWARE IS FOR DEVELOPMENT ONLY.
DISABLE IT BEFORE PUSHING TO GIT SO THAT IT
IS NOT AVAILABLE TO PRODUCTION ENVIRONMENTS

func Debug() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {

			// Try to read the body from the request
			request := ctx.Request()
			body, err := re.ReadRequestBody(request)

			if err != nil {
				return derp.Wrap(err, "middleware.Debug", "Unable to read body from request")
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
*/
