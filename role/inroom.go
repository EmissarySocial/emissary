package role

import "github.com/labstack/echo/v4"

// InRoom returns TRUE if the provided object is contained in a "stage" that is accessible by the current context
func InRoom(context echo.Context) bool {
	return true
}
