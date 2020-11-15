package handler

import (
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// PostAuthentication generates an echo.HandlerFunc that calls the Sterakno PostSignin function.
func PostAuthentication(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return err
		}

		service := factory.Steranko()

		return service.PostSignin(ctx)
	}
}
