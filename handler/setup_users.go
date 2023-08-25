package handler

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
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
		modal, err := displayDomainUsersModal(domainConfig, factory, templates)

		if err != nil {
			return derp.Wrap(err, location, "Error rendering modal")
		}

		// Wrap it as a modal
		return ctx.HTML(http.StatusOK, render.WrapModal(ctx.Response(), modal, "class:large"))
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
		user := model.NewUser()

		user.DisplayName = data.GetString("displayName")
		user.Username = data.GetString("username")
		user.EmailAddress = data.GetString("emailAddress")
		user.IsOwner = true
		user.IsPublic = true

		// Try to save the new user record
		userService := factory.User()
		if err := userService.Save(&user, "Created by Server Admin"); err != nil {
			return derp.Wrap(err, location, "Error saving user")
		}

		// Display the modal's NEW inner contents
		modal, err := displayDomainUsersModal(domainConfig, factory, templates)

		if err != nil {
			return derp.Wrap(err, location, "Error rendering modal")
		}

		return ctx.HTML(http.StatusOK, modal)
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
		if err := domainEmailService.SendWelcome(&user); err != nil {
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
		modal, err := displayDomainUsersModal(domainConfig, factory, templates)

		if err != nil {
			return derp.Wrap(err, location, "Error rendering modal")
		}

		return ctx.HTML(http.StatusOK, modal)
	}
}

func displayDomainUsersModal(domainConfig config.Domain, factory *domain.Factory, templates *template.Template) (string, error) {

	const location = "handler.displayDomainUsersModal"

	userService := factory.User()

	data := mapof.Any{
		"DomainID": domainConfig.DomainID,
		"Domain":   domainConfig.Label,
		"Users":    userService.ListOwnersAsSlice(),
	}

	var buffer bytes.Buffer

	if err := templates.ExecuteTemplate(&buffer, "users.html", data); err != nil {
		return "", derp.Wrap(err, location, "Error executing template")
	}

	return buffer.String(), nil
}
