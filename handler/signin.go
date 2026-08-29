package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// GetSignIn generates an echo.HandlerFunc that handles GET /signin requests
func GetSignIn(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	return executeDomainTemplate(ctx, factory, session, "user-signin")
}

// PostSignIn generates an echo.HandlerFunc that handles POST /signin requests
func PostSignIn(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	// Try to sign in using Steranko
	user, err := factory.Steranko(session).SigninFormPost(ctx)

	if err != nil {
		messageJSON, _ := json.Marshal(map[string]string{"SigninError": derp.Message(err)})
		ctx.Response().Header().Add("HX-Trigger", string(messageJSON))
		return ctx.HTML(derp.ErrorCode(err), derp.Message(err))
	}

	// If there is a "next" parameter, then redirect to that URL.  Otherwise, redirect to the user's profile.
	next := calcNextURL(ctx.QueryParam("next"))
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

// calcNextURL reduces a "next" parameter to a safe, same-origin path, avoiding open redirects and sign-in loops
func calcNextURL(next string) string {

	// If "next" is empty, then redirect to the user's profile
	if next == "" {
		return "/"
	}

	// Reduce "next" to a safe, same-origin path.  uri.PathAndQuery strips any scheme/host
	// and neutralizes protocol-relative forms, so the result can never be an open redirect
	// to another server.
	next = uri.PathAndQuery(next)

	// Do not allow redirect loops
	if strings.HasPrefix(next, "/signin") {
		return "/"
	}

	// Do not allow redirect loops
	if strings.HasPrefix(next, "/signout") {
		return "/"
	}

	// Allow this "next" URL redirect
	return next
}

// GetSignOut displays an HTML confirmation page after a user has been signed out of the system.
// The actual sign-out only happens in PostSignOut, so that state never changes on a GET request.
func GetSignOut(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	return executeDomainTemplate(ctx, factory, session, "user-signout")
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
	ctx.Response().Header().Add("HX-Redirect", "/signout")
	return ctx.NoContent(http.StatusNoContent)
}

// GetResetPassword displays the "reset password" form
func GetResetPassword(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	return executeDomainTemplate(ctx, factory, session, "reset-password")
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
		return derp.Wrap(err, location, "Reading form data")
	}

	// Try to load the user by username.  If the user cannot be found, the response
	// will still be sent.
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByUsernameOrEmail(session, transaction.EmailAddress, &user); err == nil {

		// RULE: The account exists but the reset email could not be delivered (SMTP down or
		// misconfigured).  Tell the member honestly instead of pointing them at an inbox that will
		// never receive it, and report loudly so the operator sees it -- this is the flow where a
		// broken mail server otherwise stays invisible until a locked-out member files a ticket.
		// NOTE: this response differs from the "not found" case only while mail is broken, which is a
		// mild account-enumeration signal accepted in exchange for not stranding locked-out members.
		if err := userService.SendPasswordResetEmail(session, &user, model.PasswordResetDurationReset); err != nil {
			derp.Report(derp.Wrap(err, location, "Sending password reset email", user.Username))
			return executeDomainTemplate(ctx, factory, session, "reset-error")
		}
	}

	// Uniform success message for both "email sent" and "user not found".
	return executeDomainTemplate(ctx, factory, session, "reset-confirm")
}

// GetResetCode displays a form (authenticated by the reset code) for resetting a user's password
func GetResetCode(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetResetCode"

	// Try to load the user by userID and resetCode
	userService := factory.User()

	user := model.NewUser()
	userID := ctx.QueryParam("userId")

	if err := userService.LoadByToken(session, userID, &user); err != nil {
		return derp.Wrap(err, location, "Loading user")
	}

	// Build the dot for the HTML response
	builder := build.NewPasswordReset(factory, session, ctx.Request(), ctx.Response(), user)

	// RULE: Each branch returns.  Without that, a User who fails the first test falls through
	// into the later ones and the handler writes a second complete document into the same
	// response.
	//
	// If the user was not found, then display an error message
	if user.IsNew() {
		return executeThemeTemplate(ctx, factory, "reset-code-invalid", builder)
	}

	// Is the reset code is valid, then display the form to reset the password
	if resetCode := ctx.QueryParam("code"); user.PasswordReset.IsValid(resetCode) {
		return executeThemeTemplate(ctx, factory, "reset-code", builder)
	}

	// If the reset code is expired, then give an "expired" message
	if user.PasswordReset.NotActive() {
		return executeThemeTemplate(ctx, factory, "reset-code-inactive", builder)
	}

	// Fall through means that the reset code is just plain wrong.
	return executeThemeTemplate(ctx, factory, "reset-code-invalid", builder)
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
		return derp.Wrap(err, location, "Reading form data")
	}

	// RULE: Ensure that passwords match
	if txn.Password != txn.Password2 {
		return derp.BadRequest(location, "Passwords do not match")
	}

	// RULE: New password must satisfy the server-side password policy (minimum length,
	// strength, and any configured breach rules). This is the authoritative check; the
	// reset-code template's minLength is only a client-side convenience.
	if err := factory.Steranko(session).ValidatePassword(txn.Password); err != nil {
		return derp.Wrap(err, location, "Password does not meet requirements")
	}

	// Try to load the user by userID and resetCode
	userService := factory.User()

	user := model.NewUser()

	if err := userService.LoadByResetCode(session, txn.UserID, txn.Code, &user); err != nil {
		return derp.Wrap(err, location, "Loading user")
	}

	// Update the user with the new password (hashed; never stored as plaintext)
	if err := factory.Steranko(session).SetPassword(&user, txn.Password); err != nil {
		return derp.Wrap(err, location, "Setting password")
	}

	// RULE: Reset codes are single-use.  Clear the code so this link cannot be replayed.
	user.PasswordReset = model.PasswordReset{}

	if err := userService.Save(session, &user, "Updated Password"); err != nil {
		return derp.Wrap(err, location, "Saving user")
	}

	// Reset the failed signin attempts for this user so that they can sign in with their new password right away.
	signinService := factory.SterankoSigninService(session)

	if err := signinService.ClearSigninAttempts(user.Username); err != nil {
		derp.Report(derp.Wrap(err, location, "Clearing signin attempts for user", user.Username))
	}

	// Try to send the password reset confirmation email.  If it fails, then log the error and move on.
	emailService := factory.Email()
	if err := emailService.SendPasswordResetConfirmation(session, &user); err != nil {
		derp.Report(derp.Wrap(err, location, "Sending password reset confirmation email to user", user.Username))
	}

	// Forward to the sign-in page with a success message
	return ctx.Redirect(http.StatusSeeOther, "/signin?message=password-reset&username="+user.Username)
}
