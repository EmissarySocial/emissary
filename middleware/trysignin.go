package middleware

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
)

func TrySignin(next echo.HandlerFunc) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		authorization := ctx.Request().Header.Get("Authorization")

		spew.Dump(authorization)

		return next(ctx)
	}
}
