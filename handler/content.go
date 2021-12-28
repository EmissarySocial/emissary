package handler

import (
	"net/http"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/first"
	"github.com/benpate/ghost/server"
	"github.com/benpate/html"
	"github.com/labstack/echo/v4"
)

func GetContentItemTypes(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetContentTypes", "Unrecognized Domain")
		}

		query := ctx.Request().URL.Query()
		endpoint := ctx.Request().Referer()
		editor := factory.ContentEditor(endpoint)
		itemTypes := editor.ItemTypes()

		txn := first.String(convert.String(query["txn"]), "new-item") // change-type should also work.
		itemID := convert.String(query["itemId"])
		place := convert.String(query["place"])
		check := convert.String(query["check"])

		// Render the HTML for the modal.
		b := html.New()

		b.Div().ID("modal")
		b.Div().Class("modal-underlay").Script("on click trigger closeModal").Close()
		b.Div().Class("modal-content")
		b.H1().InnerHTML("Add Another Section").Close()

		b.Div().Class("table")
		for _, itemType := range itemTypes {
			b.Div() // This one gets the .table styling
			b.Form("", "").Data("hx-post", endpoint).Data("hx-trigger", "click")
			b.Input("hidden", "type").Value(txn).Close()
			b.Input("hidden", "itemId").Value(itemID).Close()
			b.Input("hidden", "itemType").Value(itemType.Code)
			b.Input("hidden", "place").Value(place).Close()
			b.Input("hidden", "check").Value(check).Close()
			b.Div().InnerHTML(itemType.Label).Close()
			b.Div().InnerHTML(itemType.Description).Close()
			b.Close() // Form
			b.Close() // Div
		}
		b.Close()

		b.Div()
		b.Button().Script("on click trigger closeModal").InnerHTML("Cancel")

		b.CloseAll()

		// These two headers make it a modal
		ctx.Response().Header().Set("HX-Retarget", "aside")
		ctx.Response().Header().Set("HX-Push", "false")

		// Return results to client
		return ctx.HTML(http.StatusOK, b.String())
	}
}
