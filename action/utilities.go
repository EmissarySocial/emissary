package action

import (
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// getAuthorization extracts a model.Authorization record from the steranko.Context
func getAuthorization(ctx steranko.Context) model.Authorization {

	if claims, err := ctx.Authorization(); err == nil {

		if auth, ok := claims.(model.Authorization); ok {
			return auth
		}
	}

	return model.Authorization{}
}

func newForm(data interface{}) form.Form {
	var result form.Form

	if dataMap, ok := data.(map[string]interface{}); ok {
		result.UnmarshalMap(dataMap)
	}

	return result
}
