package service

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

type DomainEmail struct {
	serverEmail *ServerEmail
	smtp        config.SMTPConnection
	owner       config.Owner
	label       string
	hostname    string
}

func NewDomainEmail(serverEmail *ServerEmail) DomainEmail {
	return DomainEmail{
		serverEmail: serverEmail,
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *DomainEmail) Refresh(configuration config.Domain) {
	service.smtp = configuration.SMTPConnection
	service.owner = configuration.Owner
	service.label = configuration.Label
	service.hostname = configuration.Hostname
}

/*******************************
 * HARD-CODED EMAIL TEMPLATES
 *******************************/

// SendWelcome sends a welcome email to the user.  This method
// returns an error so that it CAN NOT be run asynchronously.
func (service *DomainEmail) SendWelcome(user *model.User) error {

	// Send the welcome email
	err := service.serverEmail.Send(
		service.smtp,
		"user-welcome",
		service.owner.EmailAddress,
		[]string{user.EmailAddress},
		"Welcome to Emissary",
		mapof.Any{
			// User info available to the template
			"UserID":      user.UserID.Hex(),
			"Username":    user.Username,
			"DisplayName": user.DisplayName,
			"ResetCode":   user.PasswordReset.AuthCode,
			"ExpireDate":  user.PasswordReset.ExpireDate,

			// Domain info available to the template
			"Owner": service.owner,
			"Host":  service.host(),
			"Label": service.label,
		},
	)

	return derp.Wrap(err, "service.DomainEmail.SendWelcome", "Error sending welcome email to user", user.EmailAddress)
}

// SendPasswordReset sends a passowrd reset email to the user.  This method
// swallows errors so that it can be run asynchronously.
func (service *DomainEmail) SendPasswordReset(user *model.User) error {

	// Send the welcome email
	err := service.serverEmail.Send(
		service.smtp,
		"user-password-reset",
		service.owner.EmailAddress,
		[]string{user.EmailAddress},
		"Emissary Password Reset",
		mapof.Any{
			// User info available to the template
			"UserID":      user.UserID.Hex(),
			"Username":    user.Username,
			"DisplayName": user.DisplayName,
			"ResetCode":   user.PasswordReset.AuthCode,
			"ExpireDate":  user.PasswordReset.ExpireDate,

			// Domain info available to the template
			"Owner": service.owner,
			"Host":  service.host(),
			"Label": service.label,
		},
	)

	return derp.Wrap(err, "service.DomainEmail.SendWelcome", "Error sending password reset email to user", user.Username)
}

func (service *DomainEmail) host() string {
	return domain.Protocol(service.hostname) + service.hostname
}
