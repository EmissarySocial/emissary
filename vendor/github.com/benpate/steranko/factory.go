package steranko

import "github.com/labstack/echo/v4"

// Factory is used in multi-tenant environments to locate the
// steranko instance that will be used (based on the context)
type Factory interface {

	// Steranko retrieves the correct instance to use
	// for this domain or returns an error
	Steranko(ctx echo.Context) (*Steranko, error)
}
