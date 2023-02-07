package gofed

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang-jwt/jwt/v4"
)

func getAuthorization(request *http.Request, jwtService *service.JWT) (model.Authorization, error) {

	authorization := model.NewAuthorization()

	// Try to get the JWT key from the request header
	jwtString := request.Header.Get("Authorization")

	if jwtString == "" {
		return authorization, derp.NewForbiddenError("gofed.getAuthenticatedUserID", "Missing Authorization Header")
	}

	// Try to parse the JWT from the header string
	token, err := jwt.ParseWithClaims(jwtString, &authorization, jwtService.FindJWTKey)

	if err != nil {
		return authorization, derp.Wrap(err, "gofed.getAuthenticatedUserID", "Unable to parse JWT")
	}

	spew.Dump("getAuthorization", token, authorization)

	return authorization, nil
}
