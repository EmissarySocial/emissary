package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/middleware"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/server"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

func GetAction(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		_, stream, action, err := getStreamAndAction(factoryManager, ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.PostAction", "Error")
		}

		authorization := getAuthorization(ctx)

		result, err := action.Get(stream, authorization)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.DeleteStream", "Error executing Action")
		}

		return ctx.HTML(http.StatusOK, result)
	}
}

func PostAction(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		_, stream, action, err := getStreamAndAction(factoryManager, ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.PostAction", "Error")
		}

		authorization := getAuthorization(ctx)

		result, err := action.Post(stream, authorization)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.DeleteStream", "Error executing Action")
		}

		return ctx.HTML(http.StatusOK, result)
	}
}

func getAuthorization(ctx echo.Context) model.Authorization {
	ghostContext := ctx.(middleware.GhostContext)
	return ghostContext.Authorization()
}

func getStreamAndAction(factoryManager *server.FactoryManager, ctx echo.Context) (*service.Stream, *model.Stream, model.Action, error) {

	var actionID string

	factory, err := factoryManager.ByContext(ctx)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, "ghost.handler.getStreamAndAction", "Unrecognized Domain")
	}

	streamService := factory.Stream()

	stream, err := streamService.LoadByToken(ctx.Param("stream"))

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, "ghost.handler.getStreamAndAction", "Error Loading Stream")
	}

	templateService := factory.Template()

	template, err := templateService.Load(stream.TemplateID)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, "ghost.handler.getStreamAndAction", "Error Loading Template")
	}

	if ctx.Request().Method == http.MethodDelete {
		actionID = "delete"
	} else {
		actionID = ctx.Param("action")
	}

	action, ok := template.Actions[actionID]

	if !ok {
		return nil, nil, nil, derp.New(derp.CodeInternalError, "ghost.handler.getStreamAndAction", "Invalid Action")
	}

	// Verify authorization
	ghostContext := ctx.(middleware.GhostContext)
	authorization := ghostContext.Authorization()

	if !action.UserCan(stream, authorization) {
		return nil, nil, nil, derp.New(derp.CodeForbiddenError, "ghost.handler.getStreamAndAction", "Forbidden")
	}

	return streamService, stream, action, nil

}
