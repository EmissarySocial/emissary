package scope

import (
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// NotDeleted filters out all records that have not been "virtually deleted" from the database.
func NotDeleted(ctx echo.Context) (expression.Expression, *derp.Error) {
	return data.Expression{{"journal.deleteDate", expression.OperatorEqual, 0}}, nil
}
