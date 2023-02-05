package gofed

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// getSignedInUserID returns the UserID for the current request.
// If the authorization is not valid or not present, then the error contains http.StatusUnauthorized
func getSignedInUserID(ctx context.Context) (primitive.ObjectID, error) {

	const location = "handler.getSignedInUserID"

	/*
		sterankoContext, ok := ctx.(*steranko.Context)

		if !ok {
			return primitive.NilObjectID, derp.NewUnauthorizedError(location, "Invalid Authorization")
		}

		authorization, err := sterankoContext.Authorization()

		if err != nil {
			err = derp.Wrap(err, location, "Invalid Authorization")
			derp.SetErrorCode(err, http.StatusUnauthorized)
			return primitive.NilObjectID, err
		}

		auth, ok := authorization.(*model.Authorization)

		if !ok {
			return primitive.NilObjectID, derp.NewUnauthorizedError(location, "Invalid Authorization", authorization)
		}

		return auth.UserID, nil
	*/

	// TODO: CRITICAL: How do we determine the UserID from the context.Context ??

	return primitive.NilObjectID, nil
}
