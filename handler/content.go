package handler

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/server"
	"github.com/benpate/nebula"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

func GetAdminContentPanel(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Property Panels are only allowed for Domain Admins.
		sterankoContext := ctx.(*steranko.Context)
		if !isOwner(sterankoContext.Authorization()) {
			return derp.NewForbiddenError("ghost.handler.GetContentPropertyPanel", "Access Forbidden")
		}

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetContentPropertyPanel", "Unrecognized Domain")
		}

		var stream model.Stream
		streamService := factory.Stream()

		if err := streamService.LoadByToken(ctx.Param("param2"), &stream); err != nil {
			return derp.Wrap(err, "ghost.handler.GetContentPropertyPanel", "Error loading stream")
		}

		panel, err := nebula.Prop(factory.ContentLibrary(), &stream.Content, ctx.Request().URL.Query(), ctx.Request().Referer())

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetContentPropertyPanel", "Error generating property panel")
		}

		return ctx.HTML(200, render.WrapModal(ctx.Response().Header(), panel))
	}
}
