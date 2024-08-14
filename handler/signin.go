package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
)

// GetSignIn generates an echo.HandlerFunc that handles GET /signin requests
func GetSignIn(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Locate the current domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.NewInternalError("handler.PostSignIn", "Invalid Domain.")
		}

		// Get the standard Signin page
		template := factory.Domain().Theme().HTMLTemplate

		domain := factory.Domain().Get()

		// Get a clean version of the URL query parameters
		queryString := cleanQueryParams(ctx.QueryParams())
		queryString["domainName"] = domain.Label
		queryString["domainIcon"] = domain.IconURL()
		queryString["hasRegistrationForm"] = factory.Domain().HasRegistrationForm()

		// Render the template
		if err := template.ExecuteTemplate(ctx.Response(), "signin", queryString); err != nil {
			return derp.Wrap(err, "handler.GetSignIn", "Error executing template")
		}

		return nil
	}
}

// PostSignIn generates an echo.HandlerFunc that handles POST /signin requests
func PostSignIn(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Locate the current domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.NewInternalError("handler.PostSignIn", "Invalid Domain.")
		}

		// Try to sign in using Steranko
		s := factory.Steranko()
		if err := s.SignIn(ctx); err != nil {
			ctx.Response().Header().Add("HX-Trigger", "SigninError")
			return ctx.HTML(derp.ErrorCode(err), derp.Message(err))
		}

		// If there is a "next" parameter, then redirect to that URL.
		if next := ctx.QueryParam("next"); next != "" {
			ctx.Response().Header().Add("Hx-Redirect", next)
			return ctx.NoContent(http.StatusNoContent)
		}

		// Return a success message (and redirect on the client)
		// ctx.Response().Header().Add("Hx-Trigger", "SigninSuccess")
		ctx.Response().Header().Add("Hx-Redirect", "/@me")
		return ctx.NoContent(http.StatusNoContent)
	}
}

// PostSignOut generates an echo.HandlerFunc that handles POST /signout requests
func PostSignOut(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.PostSignOut"

	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Invalid Domain", derp.WithCode(http.StatusBadRequest))
		}

		s := factory.Steranko()

		if hasBackupProfile := s.SignOut(ctx); hasBackupProfile {
			ctx.Response().Header().Add("HX-Redirect", "/admin/users")
		} else {
			ctx.Response().Header().Add("HX-Redirect", "/")
		}

		// Forward the user back to the home page of the website.
		return ctx.NoContent(http.StatusNoContent)
	}
}

func GetResetPassword(serverFactory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return executeDomainTemplate(serverFactory, ctx, "reset-password")
	}
}

func PostResetPassword(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.PostResetPassword"

	return func(ctx echo.Context) error {

		var transaction struct {
			EmailAddress string `form:"emailAddress"`
		}

		// Try to get the POST transaction data from the request body
		if err := ctx.Bind(&transaction); err != nil {
			return derp.Wrap(err, location, "Error binding form data")
		}

		// Try to get the factory for this domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.NewInternalError(location, "Invalid domain")
		}

		// Try to load the user by username.  If the user cannot be found, the response
		// will still be sent.
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByUsernameOrEmail(transaction.EmailAddress, &user); err == nil {
			userService.SendPasswordResetEmail(&user)
		}

		// Return a success message regardless of whether or not the user was found.
		template := factory.Domain().Theme().HTMLTemplate

		if err := template.ExecuteTemplate(ctx.Response(), "reset-confirm", nil); err != nil {
			return derp.Wrap(err, location, "Error executing template")
		}

		return nil
	}
}

// GetResetCode displays a form (authenticated by the reset code) for resetting a user's password
func GetResetCode(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetResetCode"

	return func(ctx echo.Context) error {

		// Try to get the factory for this domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.NewInternalError(location, "Invalid domain")
		}

		// Try to load the user by userID and resetCode
		userService := factory.User()

		user := model.NewUser()
		userID := ctx.QueryParam("userId")
		resetCode := ctx.QueryParam("code")

		if err := userService.LoadByToken(userID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user")
		}

		// Get the template that will build the HTML response
		template := factory.Domain().Theme().HTMLTemplate
		domain := factory.Domain().Get()
		object := mapof.Any{
			"domainName": domain.Label,
			"domainIcon": domain.IconURL(),
		}

		// If the user was not found, then display an error message
		if user.IsNew() {
			if err := template.ExecuteTemplate(ctx.Response(), "reset-code-invalid", object); err != nil {
				return derp.Wrap(err, location, "Error executing template")
			}
		}

		// Is the reset code is valid, then display the form to reset the password
		if user.PasswordReset.IsValid(resetCode) {

			object["userId"] = userID
			object["username"] = user.Username
			object["displayName"] = user.DisplayName
			object["code"] = resetCode

			if err := template.ExecuteTemplate(ctx.Response(), "reset-code", object); err != nil {
				return derp.Wrap(err, location, "Error executing template")
			}

			return nil
		}

		// If the reset code is expired, then give an "expired" message
		if user.PasswordReset.NotActive() {
			if err := template.ExecuteTemplate(ctx.Response(), "reset-code-inactive", object); err != nil {
				return derp.Wrap(err, location, "Error executing template")
			}

			return nil
		}

		// Fall through means that the reset code is just plain wrong.
		if err := template.ExecuteTemplate(ctx.Response(), "reset-code-invalid", object); err != nil {
			return derp.Wrap(err, location, "Error executing template")
		}

		return nil
	}
}

func PostResetCode(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Try to get the transaction data from the request body.
		var txn struct {
			Password  string `form:"password"`
			Password2 string `form:"password2"`
			UserID    string `form:"userId"`
			Code      string `form:"code"`
		}

		if err := ctx.Bind(&txn); err != nil {
			return derp.Wrap(err, "handler.PostResetCode", "Error binding form data")
		}

		// RULE: Ensure that passwords match
		if txn.Password != txn.Password2 {
			return derp.NewBadRequestError("handler.PostResetCode", "Passwords do not match")
		}

		// Try to get the factory for this domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.NewInternalError("handler.GetResetCode", "Invalid domain")
		}

		// Try to load the user by userID and resetCode
		userService := factory.User()

		user := model.NewUser()

		if err := userService.LoadByResetCode(txn.UserID, txn.Code, &user); err != nil {
			return derp.Wrap(err, "handler.GetResetCode", "Error loading user")
		}

		// Update the user with the new password
		user.SetPassword(txn.Password)

		if err := userService.Save(&user, "Updated Password"); err != nil {
			return derp.Wrap(err, "handler.GetResetCode", "Error saving user")
		}

		// Forward to the sign-in page with a success message
		return ctx.Redirect(http.StatusSeeOther, "/signin?message=password-reset&username="+user.Username)
	}
}
