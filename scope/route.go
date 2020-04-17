package scope

import (
	"strings"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Route creates a data.Criteria based on the route parameters in the context.
func Route(ctx echo.Context) (data.Expression, *derp.Error) {

	result := data.Expression{}
	routeParams := ctx.ParamNames()
	queryParams := ctx.QueryParams()

	// Scan route parameters for valid arguments
	for _, param := range routeParams {

		value := ctx.Param(param)

		if value == "" {
			return result, derp.New(derp.CodeBadRequestError, "scope.Route", "Route parameter cannot be empty", param)
		}

		switch param {

		// These are special-case arguments that are treated as strings.
		case "username", "section", "page":

			result.Add(param, "=", value)

		// This is the default that's added to *most* routes by presto.  These are treated as bson.ObjectIDs
		case "id":

			objectID, err := primitive.ObjectIDFromHex(value)

			if err != nil {
				return result, derp.New(derp.CodeBadRequestError, "scope.Route", "Invalid objectID in route parameter", param, value)
			}

			result.Add("_id", "=", objectID)

		// Unknown route parameters should throw an error.
		default:
			return result, derp.New(derp.CodeBadRequestError, "scope.Route", "Unrecognized route parameter", param, value)
		}
	}

	// Scan query parameters for valid arguments
	for name, value := range queryParams {

		if len(value) == 0 {
			return result, derp.New(derp.CodeBadRequestError, "scope.Route", "Query parameter cannot be empty", name, value)
		}

		valueString := strings.Join(value, ",")

		switch name {
		case "contentId":

			objectID, err := primitive.ObjectIDFromHex(valueString)

			if err != nil {
				return result, derp.New(derp.CodeBadRequestError, "scope.Route", "Invalid objectID query parameter", name, valueString)
			}

			result.Add(name, data.OperatorEqual, objectID)

		// Unknown query parameters SHOULD NOT throw an error.  There may be
		// other parts of the application that are using query parameters that
		// will not factor in to this scope. (e.g. page, pageSize, etc.)
		default:
		}
	}

	return result, nil
}
