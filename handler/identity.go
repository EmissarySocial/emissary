package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/steranko"
)

func GetIdentity(ctx *steranko.Context, factory *domain.Factory, identity *model.Identity) error {

	const location = "handler.GetIdentity"

	return ctx.NoContent(http.StatusOK)
}
