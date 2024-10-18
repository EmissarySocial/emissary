package handler

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SetupDomainUsersGet(serverFactory *server.Factory, templates *template.Template) echo.HandlerFunc {

	const location = "handler.SetupDomainUsersGet"

	return func(ctx echo.Context) error {

		// Get the domain configuration
		domainID := ctx.Param("domain")
		domainConfig, factory, err := serverFactory.ByDomainID(domainID)

		if err != nil {
			return derp.Wrap(err, location, "Error loading factory")
		}

		// Display the modal's inner content
		return displayDomainUsersModal(ctx, domainConfig, factory, templates)
	}
}

func SetupDomainUserPost(serverFactory *server.Factory, templates *template.Template) echo.HandlerFunc {

	const location = "handler.SetupDomainUsersPost"

	return func(ctx echo.Context) error {

		// Collect the transaction data from the request
		data := mapof.NewAny()

		if err := ctx.Bind(&data); err != nil {
			return derp.Wrap(err, location, "Error binding data")
		}

		// Try to load the requested domain
		domainID := ctx.Param("domain")
		domainConfig, factory, err := serverFactory.ByDomainID(domainID)

		if err != nil {
			return derp.Wrap(err, location, "Error loading factory")
		}

		// Populate the new user record
		userService := factory.User()
		user := model.NewUser()

		// Special rules for local domains
		if factory.IsLocalhost() {

			// Allow admins to UPDATE domain owners (if "userId" is provided)
			if userID, err := primitive.ObjectIDFromHex(ctx.QueryParam("userId")); err == nil {

				if err := userService.LoadByID(userID, &user); err != nil {
					return derp.Wrap(err, location, "Error loading user")
				}
			}

			// Allow admins to set passwords
			if password := data.GetString("password"); password != "" {
				sterankoService := factory.Steranko()
				if err := sterankoService.SetPassword(&user, password); err != nil {
					return derp.Wrap(err, location, "Error setting password")
				}
			}
		}

		// Populate the User record with the new data
		user.DisplayName = data.GetString("displayName")
		user.Username = data.GetString("username")
		user.EmailAddress = data.GetString("emailAddress")
		user.IsOwner = true
		user.IsPublic = true

		// Try to save the new user record
		if err := userService.Save(&user, "Created by Server Admin"); err != nil {
			return derp.Wrap(err, location, "Error saving user")
		}

		// Set the query parameter to display the updated user
		ctx.QueryParams().Set("userId", user.UserID.Hex())

		// Display the modal's NEW inner contents
		return displayDomainUsersModal(ctx, domainConfig, factory, templates)
	}
}

func SetupDomainUserInvite(serverFactory *server.Factory, templates *template.Template) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		domainID := ctx.Param("domain")
		_, factory, err := serverFactory.ByDomainID(domainID)

		if err != nil {
			return derp.Wrap(err, "handler.SetupDomainUserInvite", "Error loading factory")
		}

		// Try to load the requested User
		user := model.NewUser()
		userID := ctx.Param("user")
		userService := factory.User()

		if err := userService.LoadByToken(userID, &user); err != nil {
			return derp.Wrap(err, "handler.SetupDomainUserInvite", "Error loading user")
		}

		// Try to (re?)send the email invitation
		domainEmailService := factory.Email()
		if err := domainEmailService.SendPasswordReset(&user); err != nil {
			return derp.Wrap(err, "handler.SetupDomainUserInvite", "Error sending email")
		}

		return nil
	}
}

func SetupDomainUserDelete(serverFactory *server.Factory, templates *template.Template) echo.HandlerFunc {

	const location = "handler.SetupDomainUsersPost"

	return func(ctx echo.Context) error {

		// Try to load the requested domain
		domainID := ctx.Param("domain")
		domainConfig, factory, err := serverFactory.ByDomainID(domainID)

		if err != nil {
			return derp.Wrap(err, location, "Error loading factory")
		}

		// Populate the new user record
		user := model.NewUser()

		// Try to find the existing user record
		userService := factory.User()

		if err := userService.LoadByToken(ctx.Param("user"), &user); err != nil {
			return derp.Wrap(err, location, "Error loading user")
		}

		// Try to delete the user record
		if err := userService.Delete(&user, "Deleted by Server Admin"); err != nil {
			return derp.Wrap(err, location, "Error deleting user")
		}

		// Display the modal's NEW inner contents
		return displayDomainUsersModal(ctx, domainConfig, factory, templates)
	}
}

func displayDomainUsersModal(ctx echo.Context, domainConfig config.Domain, factory *domain.Factory, templates *template.Template) error {

	const location = "handler.displayDomainUsersModal"

	// Populate the data value
	userService := factory.User()

	data := mapof.Any{
		"DomainID":  domainConfig.DomainID,
		"Domain":    domainConfig.Label,
		"Users":     userService.ListOwnersAsSlice(),
		"UpdatedID": ctx.QueryParam("userId"),
	}

	// Pick the template based on the current Domain
	filename := "users.html"

	if factory.IsLocalhost() {
		filename = "users-local.html"
	}

	// Build the modal dialog body
	var buffer bytes.Buffer
	if err := templates.ExecuteTemplate(&buffer, filename, data); err != nil {
		return derp.Wrap(err, location, "Error executing template")
	}

	// Set Headers to display modal dialog
	header := ctx.Response().Header()
	header.Set("Hx-Push-Url", "false")
	header.Set("Hx-Reswap", "innerHTML")
	header.Set("Hx-Retarget", "aside")

	// Return the HTML content to the caller
	return ctx.HTML(http.StatusOK, buffer.String())
}
