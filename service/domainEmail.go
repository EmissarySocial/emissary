package service

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

type DomainEmail struct {
	serverEmail     *ServerEmail
	domainService   *Domain
	sterankoService *steranko.Steranko
	smtp            config.SMTPConnection
	owner           config.Owner
	label           string
	hostname        string
}

func NewDomainEmail(serverEmail *ServerEmail) DomainEmail {
	return DomainEmail{
		serverEmail: serverEmail,
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *DomainEmail) Refresh(configuration config.Domain, domainService *Domain, sterankoService *steranko.Steranko) {
	service.domainService = domainService
	service.sterankoService = sterankoService
	service.smtp = configuration.SMTPConnection
	service.owner = configuration.Owner
	service.label = configuration.Label
	service.hostname = configuration.Hostname
}

/******************************************
 * Email Templates
 ******************************************/

// SendWelcome sends a welcome email to the user.  This method
// returns an error so that it CAN NOT be run asynchronously.
func (service *DomainEmail) SendWelcome(txn model.RegistrationTxn) error {

	const location = "service.DomainEmail.SendWelcome"

	// Create a JWT with the registration information, and populate it into the Token
	token, err := service.sterankoService.CreateJWT(txn.Claims())

	if err != nil {
		return derp.Wrap(err, location, "Error creating JWT")
	}

	// Get the domain information from the DomainService
	domain := service.domainService.Get()

	// Send the welcome email
	err = service.serverEmail.Send(
		service.smtp,
		service.owner,
		"user-welcome",
		"User",
		mapof.Any{
			// User info available to the template
			"Username": txn.Username,
			"Name":     txn.DisplayName,
			"Email":    txn.EmailAddress,
			"Token":    token,

			// Domain info available to the template
			"Domain_Owner": service.owner,
			"Domain_URL":   service.host(),
			"Domain_Name":  domain.Label,
			"Domain_Icon":  domain.IconURL(),
		},
	)

	if err != nil {
		return derp.Wrap(err, location, "Error sending welcome email to user", txn.EmailAddress)
	}

	// Woot!
	return nil
}

// SendPasswordReset sends a passowrd reset email to the user.  This method
// swallows errors so that it can be run asynchronously.
func (service *DomainEmail) SendPasswordReset(user *model.User) error {

	const location = "service.DomainEmail.SendPasswordReset"

	domain := service.domainService.Get()

	// Send the password reset email
	err := service.serverEmail.Send(
		service.smtp,
		service.owner,
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

			// Domain info available to the template
			"Domain_Owner": service.owner,
			"Domain_URL":   service.host(),
			"Domain_Name":  domain.Label,
			"Domain_Icon":  domain.IconURL(),
		},
	)

	if err != nil {
		return derp.Wrap(err, location, "Error sending password reset email to user", user.Username)
	}

	return nil
}

// SendGuestCode sends JWT signin code to the provided email address, which will
// sign their "Identity" into the system
func (service *DomainEmail) SendGuestCode(identifier string, token string) error {

	const location = "service.DomainEmail.SendGuestCode"

	domain := service.domainService.Get()

	// Send the welcome email
	err := service.serverEmail.Send(
		service.smtp,
		service.owner,
		"user-guest-code",
		"Identity",
		mapof.Any{
			// User info available to the template
			"Email": identifier,
			"Token": token,

			// Domain info available to the template
			"Domain_Owner": service.owner,
			"Domain_URL":   service.host(),
			"Domain_Name":  domain.Label,
			"Domain_Icon":  domain.IconURL(),
		},
	)

	if err != nil {
		return derp.Wrap(err, location, "Error sending guest code to: "+identifier)
	}

	// Woot!
	return nil
}

func (service *DomainEmail) SendFollowerConfirmation(actor model.PersonLink, follower *model.Follower) error {

	const location = "service.DomainEmail.SendFollowerConfirmation"

	domain := service.domainService.Get()

	// Send the confirmation email
	err := service.serverEmail.Send(
		service.smtp,
		service.owner,
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

			// Domain info available to the template
			"Domain_Owner": service.owner,
			"Domain_URL":   service.host(),
			"Domain_Name":  domain.Label,
			"Domain_Icon":  domain.IconURL(),
		},
	)

	if err != nil {
		return derp.Wrap(err, location, "Error sending follow confirmation email to user", follower.Actor.EmailAddress)
	}

	return nil
}

func (service *DomainEmail) SendFollowerActivity(follower *model.Follower, activity mapof.Any) error {

	const location = "service.DomainEmail.SendFollowerActivity"

	domain := service.domainService.Get()

	// Send the activity email
	err := service.serverEmail.Send(
		service.smtp,
		service.owner,
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

			// Domain info available to the template
			"Domain_Owner": service.owner,
			"Domain_URL":   service.host(),
			"Domain_Name":  domain.Label,
			"Domain_Icon":  domain.IconURL(),
			"Unsubscribe":  follower.UnsubscribeLink(domain.Host()),
		},
	)

	if err != nil {
		return derp.Wrap(err, location, "Error sending follower email to user", follower.Actor.EmailAddress)
	}

	return nil
}

func (service *DomainEmail) host() string {
	return domain.AddProtocol(service.hostname)
}
