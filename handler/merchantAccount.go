package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

func PostRefreshMerchantAccounts(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.PostRefreshMerchantAccounts"

	productService := factory.Product()

	if _, _, err := productService.SyncRemoteProducts(user.UserID); err != nil {
		return derp.Wrap(err, location, "Unable to refresh remote products for User", user.UserID)
	}

	header := ctx.Response().Header()
	header.Set("Hx-Trigger", "refreshPage")

	return ctx.NoContent(http.StatusOK)
}
