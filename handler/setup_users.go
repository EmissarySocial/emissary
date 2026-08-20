package handler

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SetupDomainUsersGet renders the list of owners for a Domain, as an htmx fragment
func SetupDomainUsersGet(serverFactory *server.SetupFactory, templates *template.Template) echo.HandlerFunc {

	const location = "handler.SetupDomainUsersGet"

	return func(ctx echo.Context) error {

		// RULE: This endpoint serves a bare htmx fragment. Rendered standalone (a direct
		// navigation) it is a scriptless page with no working controls (BUG-109), so
		// non-htmx requests are sent to the parent page instead.
		if !isHTMXRequest(ctx) {
			return ctx.Redirect(http.StatusSeeOther, "/domains")
		}

		// Get the domain configuration
		domainID := ctx.Param("domain")
		domainConfig, factory, err := serverFactory.ByDomainID(domainID)

		if err != nil {
			return derp.Wrap(err, location, "Loading factory")
		}

		// Open a database session
		session, err := factory.Server().Session(ctx.Request().Context())

		if err != nil {
			return derp.Wrap(err, location, "Opening database session")
		}

		defer session.Close()

		// Display the modal's inner content
		return displayDomainUsersModal(ctx, factory, session, domainConfig, templates)
	}
}

// SetupDomainUserPost adds or updates an owner on a Domain
func SetupDomainUserPost(serverFactory *server.SetupFactory, templates *template.Template) echo.HandlerFunc {

	const location = "handler.SetupDomainUsersPost"

	return func(ctx echo.Context) error {

		// Collect the transaction data from the request
		var data struct {
			DisplayName  string `form:"displayName"`
			EmailAddress string `form:"emailAddress"`
			Username     string `form:"username"`
			Password     string `form:"password"`
		}

		if err := ctx.Bind(&data); err != nil {
			return derp.Wrap(err, location, "Reading form data")
		}

		// Try to load the requested domain
		domainID := ctx.Param("domain")
		domainConfig, factory, err := serverFactory.ByDomainID(domainID)

		if err != nil {
			return derp.Wrap(err, location, "Loading factory")
		}

		// Open a database session
		session, err := factory.Server().Session(ctx.Request().Context())

		if err != nil {
			return derp.Wrap(err, location, "Opening database session")
		}

		defer session.Close()

		// Populate the new user record
		userService := factory.User()
		user := model.NewUser()

		// Allow admins to UPDATE domain owners (if "userId" is provided)
		if userID, err := primitive.ObjectIDFromHex(ctx.QueryParam("userId")); err == nil {

			if err := userService.LoadByID(session, userID, &user); err != nil {
				return derp.Wrap(err, location, "Loading user")
			}
		}

		// Allow admins to set passwords
		if password := data.Password; password != "" {
			if err := factory.Steranko(session).SetPassword(&user, password); err != nil {
				return derp.Wrap(err, location, "Setting password")
			}
		}

		// Populate the User record with the new data
		user.DisplayName = data.DisplayName
		user.Username = data.Username
		user.EmailAddress = data.EmailAddress
		user.IsOwner = true
		user.IsPublic = true

		// Try to save the new user record
		if err := userService.Save(session, &user, "Created by Server Admin"); err != nil {
			return derp.Wrap(err, location, "Saving user")
		}

		// Set the query parameter to display the updated user
		ctx.QueryParams().Set("userId", user.UserID.Hex())

		// Display the modal's NEW inner contents
		return displayDomainUsersModal(ctx, factory, session, domainConfig, templates)
	}
}

// SetupDomainUserInvite emails a sign-in invitation to a Domain owner
func SetupDomainUserInvite(serverFactory *server.SetupFactory, templates *template.Template) echo.HandlerFunc {

	const location = "handler.SetupDomainUserInvite"

	return func(ctx echo.Context) error {

		domainID := ctx.Param("domain")
		_, factory, err := serverFactory.ByDomainID(domainID)

		if err != nil {
			return derp.Wrap(err, location, "Loading factory")
		}

		// Open a database session
		session, err := factory.Server().Session(ctx.Request().Context())

		if err != nil {
			return derp.Wrap(err, location, "Opening database session")
		}

		defer session.Close()

		// Try to load the requested User
		user := model.NewUser()
		userID := ctx.Param("user")
		userService := factory.User()

		if err := userService.LoadByToken(session, userID, &user); err != nil {
			return derp.Wrap(err, location, "Loading user")
		}

		// RULE: Reset codes are single-use, so mint a new code in case a previous one was used or expired
		if err := userService.MakeNewPasswordResetCode(session, &user, model.PasswordResetDurationWelcome); err != nil {
			return derp.Wrap(err, location, "Making password reset code")
		}

		// Try to (re?)send the email invitation
		domainEmailService := factory.Email()
		if err := domainEmailService.SendPasswordReset(&user); err != nil {
			return derp.Wrap(err, location, "Sending email")
		}

		return nil
	}
}

// SetupDomainUserDelete removes an owner from a Domain
func SetupDomainUserDelete(serverFactory *server.SetupFactory, templates *template.Template) echo.HandlerFunc {

	const location = "handler.SetupDomainUsersPost"

	return func(ctx echo.Context) error {

		// Try to load the requested domain
		domainID := ctx.Param("domain")
		domainConfig, factory, err := serverFactory.ByDomainID(domainID)

		if err != nil {
			return derp.Wrap(err, location, "Loading factory")
		}

		// Open a database session
		session, err := factory.Server().Session(ctx.Request().Context())

		if err != nil {
			return derp.Wrap(err, location, "Opening database session")
		}

		defer session.Close()

		// Try to find the existing user record
		user := model.NewUser()
		userService := factory.User()

		if err := userService.LoadByToken(session, ctx.Param("user"), &user); err != nil {
			return derp.Wrap(err, location, "Loading user")
		}

		// Try to delete the user record
		if err := userService.Delete(session, &user, "Deleted by Server Admin"); err != nil {
			return derp.Wrap(err, location, "Deleting user")
		}

		// Display the modal's NEW inner contents
		return displayDomainUsersModal(ctx, factory, session, domainConfig, templates)
	}
}

// displayDomainUsersModal renders the owner-management modal for a Domain
func displayDomainUsersModal(ctx echo.Context, factory *service.Factory, session data.Session, domainConfig config.Domain, templates *template.Template) error {

	const location = "handler.displayDomainUsersModal"

	// Populate the data value
	userService := factory.User()

	data := mapof.Any{
		"DomainID":  domainConfig.DomainID,
		"Domain":    domainConfig.Label,
		"Users":     userService.ListOwnersAsSlice(session),
		"UpdatedID": ctx.QueryParam("userId"),
	}

	// Build the modal dialog body.  Server admins can set owner passwords on ANY domain
	// (not just localhost), so there is a single template for every domain.
	var buffer bytes.Buffer
	if err := templates.ExecuteTemplate(&buffer, "users.html", data); err != nil {
		return derp.Wrap(err, location, "Executing template")
	}

	// Set Headers to display modal dialog
	header := ctx.Response().Header()
	header.Set("Hx-Push-Url", "false")
	header.Set("Hx-Reswap", "innerHTML")
	header.Set("Hx-Retarget", "aside")

	// Return the HTML content to the caller
	return ctx.HTML(http.StatusOK, buffer.String())
}
