package middleware

/*
THIS MIDDLEWARE IS FOR DEVELOPMENT ONLY.
DISABLE IT BEFORE PUSHING TO GIT SO THAT IT
IS NOT AVAILABLE TO PRODUCTION ENVIRONMENTS

func DebugCookies() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {

			// Try to read the body from the request
			request := ctx.Request()
			fmt.Println(request.Proto + " " + request.Method + " " + dt.TrueHostname(request) + request.URL.String())
			fmt.Println("Cookies:" + strings.Join(request.Header["Cookie"], "; "))
			fmt.Println("")

			return next(ctx)
		}
	}
}
*/
