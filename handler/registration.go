package handler

import (
	"encoding/json"
	"net/http"

	"github.com/EmissarySocial/emissary/builder"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetRegister generates an echo.HandlerFunc that handles GET /register requests
func GetRegister(factoryManager *server.Factory) echo.HandlerFunc {

	const location = "handler.GetRegister"

	return func(ctx echo.Context) error {

		// Try to load the factory and domain
		factory, domain, err := loadFactoryAndDomain(factoryManager, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// If the signup form is not active, then this is a "not found" error
		if !domain.HasRegistrationForm() {
			return ctx.NoContent(http.StatusNotFound)
		}

		// Retrieve the signup template
		registrationService := factory.Registration()
		registration, err := registrationService.Load(domain.SignupID)

		if err != nil {
			return derp.Wrap(err, location, "Error loading signup template")
		}

		actionID := getActionID(ctx)

		b, err := builder.NewRegistration(factory, ctx.Request(), ctx.Response(), registration, actionID)

		if err != nil {
			return derp.Wrap(err, location, "Error creating Builder")
		}

		if err := buildHTML(factory, ctx, b, builder.ActionMethodGet); err != nil {
			return derp.Wrap(err, location, "Error building HTML")
		}

		return nil
	}
}

// PostRegister generates an echo.HandlerFunc that handles POST /register requests
func PostRegister(factoryManager *server.Factory) echo.HandlerFunc {

	const location = "handler.PostRegister"

	return func(ctx echo.Context) error {

		// Try to load the factory and domain
		factory, domain, err := loadFactoryAndDomain(factoryManager, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// If the signup form is not active, then this is a "not found" error
		if !domain.HasRegistrationForm() {
			return ctx.NoContent(http.StatusNotFound)
		}

		// Validate User Input
		userService := factory.User()

		transaction := struct {
			DisplayName string `form:"displayName"`
			Username    string `form:"username"`
			Password    string `form:"password"`
		}{}

		if err := ctx.Bind(&transaction); err != nil {
			return derp.Wrap(err, location, "Error binding user input")
		}

		errorMessages := map[string]string{}
		user := model.NewUser()

		// Validate Username is Unique
		if err := userService.LoadByUsername(transaction.Username, &user); err == nil {
			errorMessages["username"] = "Pick a different username.  This one is already in use."
		} else if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error searching for username")
		}

		// Otherwise, we got a 404 error, which is actually what we want here.
		// It means that the username is unique.

		// TODO: MEDIUM: Other validations here? Password quality?

		// Report errors
		if len(errorMessages) > 0 {
			event := map[string]any{"eventValidatorError": errorMessages}
			eventBytes, _ := json.Marshal(event)
			ctx.Response().Header().Add("HX-Trigger", string(eventBytes))
			return ctx.NoContent(http.StatusOK)
		}

		// Try to save the new user record
		user.DisplayName = transaction.DisplayName
		user.GroupIDs = []primitive.ObjectID{domain.SignupForm.GroupID}
		user.SetUsername(transaction.Username)
		user.SetPassword(transaction.Password)

		if err := userService.Save(&user, "Created by signup form"); err != nil {
			return derp.Wrap(err, location, "Error saving new user record")
		}

		// Try to sign-in with the new user's account
		s := factory.Steranko()

		cookie, err := s.CreateCertificate(ctx.Request(), &user)

		if err != nil {
			return derp.Wrap(err, location, "Error signing in user")
		}

		ctx.SetCookie(&cookie)

		// Report success to the client
		ctx.Response().Header().Add("HX-Trigger", "RegistrationSuccess")
		return ctx.NoContent(200)
	}
}
