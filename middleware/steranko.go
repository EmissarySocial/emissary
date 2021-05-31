package middleware

import (
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/server"
	"github.com/benpate/steranko"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
)

type GhostContext struct {
	steranko.Context
	authorization *model.Authorization
}

// TODO: populate authorization
func (ctx GhostContext) Authorization() model.Authorization {

	if ctx.authorization == nil {
		ctx.authorization = &model.Authorization{}
	}

	return *ctx.authorization
}

/*
// Adapter to steranko.Middleware
func Steranko(factoryManager *server.FactoryManager) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			factory, err := factoryManager.ByContext(ctx)

			if err != nil {
				return err
			}

			s := factory.Steranko()

			return s.Middleware(false)(next)(ctx)
		}
	}
}
*/

func Steranko(factoryManager *server.FactoryManager) echo.MiddlewareFunc {

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(ctx echo.Context) error {

			factory, _ := factoryManager.ByContext(ctx)
			keyService := factory.Key()

			// name := cookieName(ctx)
			name := "Authorization"

			if cookie, err := ctx.Cookie(name); err == nil {

				// claims := s.UserService.NewClaims()
				claims := model.JWTClaims{}

				if token, err := jwt.ParseWithClaims(cookie.Value, &claims, keyService.FindJWTKey); err == nil {

					// TODO: Token Expiration / Renewal
					// TODO: Errors on failed token parsing?

					return next(GhostContext{
						Context: steranko.Context{
							Context: ctx,
							Token:   token,
						},
					})
				}
			}

			return next(ctx)
		}
	}
}
