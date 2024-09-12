package handler

import (
	"encoding/json"
	"net/http"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetRegister generates an echo.HandlerFunc that handles GET /register requests
func GetRegister(ctx *steranko.Context, factory *domain.Factory, domain *model.Domain, registration *model.Registration) error {

	const location = "handler.GetRegister"

	// If the user is already signed in, then just forward to their home page
	if authorization := getAuthorization(ctx); authorization.IsAuthenticated() {
		return ctx.Redirect(http.StatusFound, "/@me")
	}

	// Build the registration form
	actionID := getActionID(ctx)

	b, err := build.NewRegistration(factory, ctx.Request(), ctx.Response(), registration, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Error creating Builder")
	}

	if err := build.AsHTML(factory, ctx, b, build.ActionMethodGet); err != nil {
		return derp.Wrap(err, location, "Error building HTML")
	}

	return nil
}

// PostRegister generates an echo.HandlerFunc that handles POST /register requests
func PostRegister(ctx *steranko.Context, factory *domain.Factory, domain *model.Domain, registration *model.Registration) error {

	const location = "handler.PostRegister"

	// Collect user input
	txn := model.NewRegistrationTxn()

	if err := ctx.Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Error binding user input")
	}

	// Validate the Transaction
	secret := domain.RegistrationData.GetString("secret")
	if !txn.IsValid(secret) {
		return derp.NewBadRequestError(location, "Invalid Registration Transaction", txn)
	}

	// Register the new User
	registrationService := factory.Registration()
	user, err := registrationService.Register(factory.Group(), factory.User(), domain, txn)

	if err != nil {
		event := map[string]any{"eventValidatorError": "Could not register this account. Please try again."}
		eventBytes, _ := json.Marshal(event)
		ctx.Response().Header().Add("HX-Trigger", string(eventBytes))
		return ctx.NoContent(http.StatusOK)
	}

	// Try to sign-in with the new user's account
	s := factory.Steranko()
	cookie, err := s.CreateCertificate(ctx.Request(), &user)

	if err != nil {
		return derp.Wrap(err, location, "Error signing in user")
	}

	ctx.SetCookie(&cookie)

	// Build confirmation response
	b, err := build.NewRegistration(factory, ctx.Request(), ctx.Response(), registration, "confirm")

	if err != nil {
		return derp.Wrap(err, location, "Error creating Builder")
	}

	if err := build.AsHTML(factory, ctx, b, build.ActionMethodGet); err != nil {
		return derp.Wrap(err, location, "Error building HTML")
	}

	// Report success to the client
	return nil
}

// PostUpdateRegistration generates an echo.HandlerFunc that handles POST /register requests
func PostUpdateRegistration(ctx *steranko.Context, factory *domain.Factory, domain *model.Domain, registration *model.Registration) error {

	const location = "handler.PostRegister"

	// Collect User info
	userInfo := struct {
		Source   string `json:"source"`
		SourceID string `json:"sourceId"`
	}{}

	if err := ctx.Bind(&userInfo); err != nil {
		return derp.Wrap(err, location, "Error binding user input")
	}

	// Collect transaction info
	txn := model.NewRegistrationTxn()

	if err := ctx.Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Error binding user input")
	}

	// Validate the Transaction
	secret := domain.RegistrationData.GetString("secret")

	if secret == "" {
		return derp.NewNotFoundError(location, "Secret not found")
	}

	if !txn.IsValid(secret) {
		return derp.NewBadRequestError(location, "Invalid Registration Transaction", txn)
	}

	// Update the User' registration
	registrationService := factory.Registration()

	if err := registrationService.UpdateRegistration(factory.Group(), factory.User(), domain, userInfo.Source, userInfo.SourceID, txn); err != nil {
		return derp.Wrap(err, location, "Error updating user registration")
	}

	return ctx.NoContent(200)
}
