package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// GetFolder renders a Folder in HTML
func GetFolder(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetFolder", "Unrecognized domain"))
		}

		token := ctx.Param("folder")

		folderService := factory.Folder()

		folder, err := folderService.LoadByToken(token)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetFolder", "Unable to load folder", token))
		}

		renderer := factory.FolderRenderer(*folder, getFolderView(ctx))
		result, err := renderPage(factory.Layout(), renderer, isFullPageRequest(ctx))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetFolder", "Error rendering folder"))
		}

		return ctx.HTML(http.StatusOK, result)
	}
}

func getFolderView(ctx echo.Context) string {

	if ctx.Request().Header.Get("hx-request") == "true" {
		return "folder-partial"
	}

	return "folder-full"
}
