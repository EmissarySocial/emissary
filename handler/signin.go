package handler

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

// GetSignIn generates an echo.HandlerFunc that handles GET /signin requests
func GetSignIn(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	// Get the standard Signin page
	template := factory.Domain().Theme().HTMLTemplate

	domain := factory.Domain().Get()

	// Get a clean version of the URL query parameters
	data := cleanQueryParams(ctx.QueryParams())
	data["DomainName"] = domain.Label
	data["DomainIcon"] = domain.IconURL()
	data["DomainImage"] = domain.ImageURL()
	data["HasRegistrationForm"] = factory.Domain().HasRegistrationForm()
	data["Next"] = url.QueryEscape(data.GetString("next"))

	// Render the template
	if err := template.ExecuteTemplate(ctx.Response(), "user-signin", data); err != nil {
		return derp.Wrap(err, "handler.GetSignIn", "Error executing template")
	}

	return nil
}

// PostSignIn generates an echo.HandlerFunc that handles POST /signin requests
func PostSignIn(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	// Try to sign in using Steranko
	user, err := factory.Steranko(session).SigninFormPost(ctx)

	if err != nil {
		ctx.Response().Header().Add("HX-Trigger", "SigninError")
		return ctx.HTML(derp.ErrorCode(err), derp.Message(err))
	}

	// If there is a "next" parameter, then redirect to that URL.  Otherwise, redirect to the user's profile.
	next := first.String(ctx.QueryParam("next"), "/@me")
	ctx.Response().Header().Add("Hx-Redirect", next)

	// Add user's Activity Intent data to the response.
	if user, isAlwaysOK := user.(*model.User); isAlwaysOK {

		message := mapof.Any{"signin-account": user.ActivityIntentProfile()}

		if messageJSON, err := json.Marshal(message); err == nil {
			ctx.Response().Header().Add("Hx-Trigger", string(messageJSON))
		}
	}

	/// 3..2..1.. Go!
	return ctx.NoContent(http.StatusNoContent)
}

// PostSignOut generates an echo.HandlerFunc that handles POST /signout requests
func PostSignOut(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	s := factory.Steranko(session)

	// If we have an admin "backup profile" then return to the admin section
	if hasBackupProfile := s.SignOut(ctx); hasBackupProfile {
		ctx.Response().Header().Add("HX-Redirect", "/admin/users")
		return ctx.NoContent(http.StatusNoContent)
	}

	// If there's a "next" parameter, then try to redirect there
	if next := ctx.QueryParam("next"); next != "" {

		// If this is a valiid URL, then redirect to the path portion only (to avoid open redirects)
		if nextURL, err := url.Parse(next); err == nil {
			ctx.Response().Header().Add("Hx-Redirect", "/signin?next="+url.QueryEscape(nextURL.Path))
			return ctx.NoContent(http.StatusNoContent)
		}
	}

	// Otherwise, just redirect to the home page.
	ctx.Response().Header().Add("HX-Redirect", "/")
	return ctx.NoContent(http.StatusNoContent)
}

// GetResetPassword displays the "reset password" form
func GetResetPassword(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	return executeDomainTemplate(ctx, factory, "reset-password")
}

// PostResetPassword processes the "reset password" form.  If the user enters a valid email address,
// then a password reset email is sent to that address.
func PostResetPassword(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostResetPassword"

	var transaction struct {
		EmailAddress string `form:"emailAddress"`
	}

	// Try to get the POST transaction data from the request body
	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Unable to read form data")
	}

	// Try to load the user by username.  If the user cannot be found, the response
	// will still be sent.
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByUsernameOrEmail(session, transaction.EmailAddress, &user); err == nil {
		userService.SendPasswordResetEmail(session, &user)
	}

	// Return a success message regardless of whether or not the user was found.
	template := factory.Domain().Theme().HTMLTemplate

	if err := template.ExecuteTemplate(ctx.Response(), "reset-confirm", nil); err != nil {
		return derp.Wrap(err, location, "Error executing template")
	}

	return nil
}

// GetResetCode displays a form (authenticated by the reset code) for resetting a user's password
func GetResetCode(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetResetCode"

	// Try to load the user by userID and resetCode
	userService := factory.User()

	user := model.NewUser()
	userID := ctx.QueryParam("userId")
	resetCode := ctx.QueryParam("code")

	if err := userService.LoadByToken(session, userID, &user); err != nil {
		return derp.Wrap(err, location, "Unable to load user")
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

// PostResetCode processes the "reset code" form to update the user's password.
// If the reset code is valid, then the user's password is updated.
func PostResetCode(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostResetCode"

	// Try to get the transaction data from the request body.
	var txn struct {
		Password  string `form:"password"`
		Password2 string `form:"password2"`
		UserID    string `form:"userId"`
		Code      string `form:"code"`
	}

	if err := ctx.Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Unable to read form data")
	}

	// RULE: Ensure that passwords match
	if txn.Password != txn.Password2 {
		return derp.BadRequestError(location, "Passwords do not match")
	}

	// Try to load the user by userID and resetCode
	userService := factory.User()

	user := model.NewUser()

	if err := userService.LoadByResetCode(session, txn.UserID, txn.Code, &user); err != nil {
		return derp.Wrap(err, location, "Unable to load user")
	}

	// Update the user with the new password
	user.SetPassword(txn.Password)

	if err := userService.Save(session, &user, "Updated Password"); err != nil {
		return derp.Wrap(err, location, "Unable to save user")
	}

	// Forward to the sign-in page with a success message
	return ctx.Redirect(http.StatusSeeOther, "/signin?message=password-reset&username="+user.Username)
}
