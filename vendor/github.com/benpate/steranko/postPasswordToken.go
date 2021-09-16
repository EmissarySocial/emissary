package steranko

import (
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// PostPasswordToken implements the http.HandlerFunc signature, and should
// be wired in to your REST API to allow users to tell the server that they
// forgot their password.  This should initiate some way for the system to
// send them a one time token to create a new password.
func (s *Steranko) PostPasswordToken(ctx echo.Context) error {

	txn := SigninTransaction{}

	if err := ctx.Bind(&txn); err != nil {
		return derp.Report(derp.Wrap(err, "steranko.PostPasswordToken", "Error binding transaction parameters"))
	}

	user := s.UserService.New()

	if err := s.UserService.Load(txn.Username, user); err != nil {

		if derp.NotFound(err) {
			return derp.New(CodeUnauthorized, "steranko.PostPasswordToken", "Unauthorized")
		}

		return derp.Wrap(err, "steranko.PostPasswordToken", "Error loading User account", txn.Username)
	}

	if err := s.UserService.RequestPasswordReset(user); err != nil {
		return derp.Wrap(err, "steranko.PostPasswordToken", "Error sending reset invitation")
	}

	return nil
}
