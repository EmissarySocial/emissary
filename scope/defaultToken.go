package scope

import (
	"github.com/benpate/data/expression"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// String generates a presto.ScoperFunc using the values provided.  Every context parameter will be compared with an "equals" comparison scope.
func String(values ...string) presto.ScoperFunc {

	return func(ctx echo.Context) (expression.Expression, *derp.Error) {

		token, err := ctx.Get("token")

		if err != nil {
			return nil, derp.New(500, "scope.DefaultToken", "Can't Read parameter 'token")
		}

		if token == "" {
			return nil, derp.New(500, "scope.DefaultToken", "Token cannot be empty")
		}

		return expression.New("token", expression.OperatorEqual, token)
	}
}
