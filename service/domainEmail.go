package service

import (
	"maps"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// DomainEmail sends the transactional emails that a single Domain generates
type DomainEmail struct {
	serverEmail   *ServerEmail
	domainService *Domain
	smtp          config.SMTPConnection
	owner         config.Owner
	label         string
	hostname      string
	newSteranko   func(session data.Session) *steranko.Steranko
}

// NewDomainEmail returns an empty DomainEmail service, which Refresh populates
func NewDomainEmail() DomainEmail {
	return DomainEmail{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates this service with the latest configuration values
func (service *DomainEmail) Refresh(factory *Factory) {
	service.serverEmail = factory.ServerEmail()
	service.domainService = factory.Domain()
	service.smtp = factory.config.SMTPConnection
	service.owner = factory.config.Owner
	service.label = factory.config.Label
	service.hostname = factory.config.Hostname
	service.newSteranko = factory.Steranko
}

/******************************************
 * Send API
 ******************************************/

// IsConfigured returns TRUE if this Domain has an SMTP connection that Send can use
func (service *DomainEmail) IsConfigured() bool {
	return !service.smtp.IsNil()
}

// Send renders the named email and delivers it over this Domain's SMTP connection
func (service *DomainEmail) Send(emailID string, data mapof.Any) error {

	const location = "service.DomainEmail.Send"

	// RULE: an unconfigured SMTP connection is a failure, not a silent success.  Returning nil
	// here would let a caller -- a contact form, say -- accept a message, deliver nothing, and
	// leave no record that it happened.  Callers that must survive this check IsConfigured first.
	if !service.IsConfigured() {
		return derp.Internal(location, "Cannot send email because SMTP is not configured for this domain", emailID)
	}

	// Copy the caller's data so that a caller reusing its map does not accumulate our additions
	values := make(mapof.Any, len(data)+4)
	maps.Copy(values, data)

	// Add the Domain values that every email template may reference
	domain := service.domainService.Get()

	values["Domain_Owner"] = service.owner
	values["Domain_URL"] = service.host()
	values["Domain_Name"] = domain.Label
	values["Domain_Icon"] = domain.IconURL()

	if err := service.serverEmail.Send(service.smtp, service.owner, emailID, values); err != nil {
		return derp.Wrap(err, location, "Sending email", emailID)
	}

	// Neither snow nor rain nor heat nor gloom of night
	return nil
}

// sendModel delivers the named email, but only if it is defined for modelName.  Each sender below
// builds a fixed data shape for a fixed email, so a definition that an administrator overrode on
// disk for some other object would render nothing but missing values.  Emails named by a Template
// need no such check: their data comes from the same file that names the email.
func (service *DomainEmail) sendModel(emailID string, modelName string, data mapof.Any) error {

	const location = "service.DomainEmail.sendModel"

	if err := service.serverEmail.RequireModel(emailID, modelName); err != nil {
		return derp.Wrap(err, location, "Email is not defined for this model object", emailID, modelName)
	}

	return service.Send(emailID, data)
}

/******************************************
 * Email Templates
 ******************************************/

// SendWelcome sends a welcome email to the user.  This method
// returns an error so that it CAN NOT be run asynchronously.
func (service *DomainEmail) SendWelcome(session data.Session, txn model.RegistrationTxn) error {

	const location = "service.DomainEmail.SendWelcome"

	// Create a JWT with the registration information, and populate it into the Token
	sterankoService := service.newSteranko(session)
	token, err := sterankoService.CreateJWT(txn.Claims())

	if err != nil {
		return derp.Wrap(err, location, "Creating JWT")
	}

	// Send the welcome email
	if err := service.sendModel(
		"user-welcome",
		"User",
		mapof.Any{
			// User info available to the template
			"Username": txn.Username,
			"Name":     txn.DisplayName,
			"Email":    txn.EmailAddress,
			"Token":    token,
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending welcome email to user", txn.EmailAddress)
	}

	// Woot!
	return nil
}

// SendPasswordReset sends a passowrd reset email to the user.  This method
// swallows errors so that it can be run asynchronously.
func (service *DomainEmail) SendPasswordReset(user *model.User) error {

	const location = "service.DomainEmail.SendPasswordReset"

	// Send the password reset email
	if err := service.sendModel(
		"user-password-reset",
		"User",
		mapof.Any{
			// User info available to the template
			"UserID":     user.UserID.Hex(),
			"Username":   user.Username,
			"Name":       user.DisplayName,
			"Email":      user.EmailAddress,
			"ResetCode":  user.PasswordReset.AuthCode,
			"ExpireDate": user.PasswordReset.ExpireDate,
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending password reset email to user", user.Username)
	}

	return nil
}

// SendPasswordResetConfirmation sends a password reset confirmation email to the user.  This method
// swallows errors so that it can be run asynchronously.
func (service *DomainEmail) SendPasswordResetConfirmation(session data.Session, user *model.User) error {

	const location = "service.DomainEmail.SendPasswordResetConfirmation"

	// Send the password reset confirmation email
	if err := service.sendModel(
		"user-password-reset-confirmation",
		"User",
		mapof.Any{
			// User info available to the template
			"UserID":     user.UserID.Hex(),
			"Username":   user.Username,
			"Name":       user.DisplayName,
			"Email":      user.EmailAddress,
			"ResetCode":  user.PasswordReset.AuthCode,
			"ExpireDate": user.PasswordReset.ExpireDate,
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending password reset confirmation email to user", user.Username)
	}

	return nil
}

// SendUserLockout sends a lockout notification email to the user.  This method
// swallows errors so that it can be run asynchronously.
func (service *DomainEmail) SendUserLockout(session data.Session, user *model.User) error {

	const location = "service.DomainEmail.SendUserLockout"

	// Send the lockout email
	if err := service.sendModel(
		"user-lockout",
		"User",
		mapof.Any{
			// User info available to the template.
			// NOTE: no ResetCode/ExpireDate -- a lockout no longer resets the password.
			"UserID":   user.UserID.Hex(),
			"Username": user.Username,
			"Name":     user.DisplayName,
			"Email":    user.EmailAddress,
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending user lockout email to user", user.Username)
	}

	return nil
}

// SendGuestCode sends JWT signin code to the provided email address, which will
// sign their "Identity" into the system
func (service *DomainEmail) SendGuestCode(identifier string, token string) error {

	const location = "service.DomainEmail.SendGuestCode"

	// Send the welcome email
	if err := service.sendModel(
		"user-guest-code",
		"Identity",
		mapof.Any{
			// User info available to the template
			"Email": identifier,
			"Token": token,
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending guest code to: "+identifier)
	}

	// Woot!
	return nil
}

// SendFollowerConfirmation emails a new Follower the link that confirms their subscription
func (service *DomainEmail) SendFollowerConfirmation(actor model.PersonLink, follower *model.Follower) error {

	const location = "service.DomainEmail.SendFollowerConfirmation"

	// Send the confirmation email
	if err := service.sendModel(
		"follower-confirmation",
		"Follower",
		mapof.Any{
			// Parent info available to the template
			"Actor": actor,

			// Follower info available to the template
			"FollowerID": follower.FollowerID.Hex(),
			"Name":       follower.Actor.Name,
			"Email":      follower.Actor.EmailAddress,
			"Secret":     follower.Data.GetString("secret"),
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending follow confirmation email to user", follower.Actor.EmailAddress)
	}

	return nil
}

// SendFollowerActivity emails an activity to a Follower who subscribed by email
func (service *DomainEmail) SendFollowerActivity(follower *model.Follower, activity mapof.Any) error {

	const location = "service.DomainEmail.SendFollowerActivity"

	domain := service.domainService.Get()

	// Send the activity email
	if err := service.sendModel(
		"follower-activity",
		"Follower",
		mapof.Any{

			// Parent info available to the template
			"ParentLink": follower.ParentURL(domain.Host()),

			// Follower info available to the template
			"FollowerID": follower.FollowerID.Hex(),
			"Name":       follower.Actor.Name,
			"Email":      follower.Actor.EmailAddress,
			"Secret":     follower.Data.GetString("secret"),

			// Activity info available to the template
			"Activity": activity,

			// Unsubscribe links available to the template (D18: two forms, two consumers)
			"Unsubscribe":             follower.UnsubscribeLink(domain.Host()),
			"UnsubscribeWithBrackets": follower.UnsubscribeLinkWithBrackets(domain.Host()),
		},
	); err != nil {
		return derp.Wrap(err, location, "Sending follower email to user", follower.Actor.EmailAddress)
	}

	return nil
}

// host returns this Domain's hostname, with its protocol prefix
func (service *DomainEmail) host() string {
	return uri.PrependProtocol(service.hostname)
}
