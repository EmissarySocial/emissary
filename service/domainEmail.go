package service

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/maps"
)

type DomainEmail struct {
	serverEmail *ServerEmail
	smtp        config.SMTPConnection
	owner       config.Owner
	label       string
	hostname    string
}

func NewDomainEmail(serverEmail *ServerEmail, configuration config.Domain) DomainEmail {

	service := DomainEmail{
		serverEmail: serverEmail,
		smtp:        configuration.SMTPConnection,
		owner:       configuration.Owner,
		label:       configuration.Label,
		hostname:    configuration.Hostname,
	}

	service.Refresh(configuration)

	return service
}

/*******************************************
 * LIFECYCLE METHODS
 *******************************************/

func (service *DomainEmail) Refresh(configuration config.Domain) {

	// Add configuration to the service
	service.label = configuration.Label
	service.hostname = configuration.Hostname
	service.smtp = configuration.SMTPConnection
	service.owner = configuration.Owner
}

/*******************************
 * HARD-CODED EMAIL TEMPLATES
 *******************************/

// SendWelcome sends a welcome email to the user.  This method
// returns an error so that it CAN NOT be run asynchronously.
func (service *DomainEmail) SendWelcome(user model.User) error {

	// Send the welcome email
	err := service.serverEmail.Send(
		service.smtp,
		"user-welcome",
		service.owner.EmailAddress,
		[]string{user.Username},
		"Welcome to Emissary",
		maps.Map{
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

	return derp.Wrap(err, "service.DomainEmail.SendWelcome", "Error sending welcome email to user", user.Username)
}

// SendPasswordReset sends a passowrd reset email to the user.  This method
// swallows errors so that it can be run asynchronously.
func (service *DomainEmail) SendPasswordReset(user model.User) {
	// Send the welcome email
	err := service.serverEmail.Send(
		service.smtp,
		"user-password-reset",
		service.owner.EmailAddress,
		[]string{user.Username},
		"Emissary Password Reset",
		maps.Map{
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

	derp.Report(derp.Wrap(err, "service.DomainEmail.SendWelcome", "Error sending welcome email to user", user.Username))
}

func (service *DomainEmail) host() string {
	return domain.Protocol(service.hostname) + service.hostname
}
