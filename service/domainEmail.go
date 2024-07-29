package service

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	mail "github.com/xhit/go-simple-mail/v2"
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

/******************************************
 * Email Templates
 ******************************************/

// SendWelcome sends a welcome email to the user.  This method
// returns an error so that it CAN NOT be run asynchronously.
func (service *DomainEmail) SendWelcome(user *model.User) error {

	fromAddress := service.owner.DisplayName + " <" + service.owner.EmailAddress + ">"

	// Build the email message
	message := mail.NewMSG().
		SetSubject("Welcome to " + service.label).
		SetFrom(fromAddress).
		// SetSender(service.owner.EmailAddress). // Don't need this if we're using 'From' header
		AddTo(user.EmailAddress)

	// Send the welcome email
	err := service.serverEmail.Send(
		service.smtp,
		message,
		"user-welcome",
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

	if err != nil {
		return derp.Wrap(err, "service.DomainEmail.SendWelcome", "Error sending welcome email to user", user.EmailAddress)
	}

	return nil
}

// SendPasswordReset sends a passowrd reset email to the user.  This method
// swallows errors so that it can be run asynchronously.
func (service *DomainEmail) SendPasswordReset(user *model.User) error {

	// Build the email message
	message := mail.NewMSG().
		SetSubject("Password Reset from " + service.host()).
		SetSender(service.owner.EmailAddress).
		AddTo(user.EmailAddress)

	// Send the password reset email
	err := service.serverEmail.Send(
		service.smtp,
		message,
		"user-password-reset",
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

	if err != nil {
		return derp.Wrap(err, "service.DomainEmail.SendWelcome", "Error sending password reset email to user", user.Username)
	}

	return nil
}

func (service *DomainEmail) SendFollowerConfirmation(actor model.PersonLink, follower *model.Follower) error {

	// Build the email message
	message := mail.NewMSG().
		SetSubject("Please Confirm Email Updates from " + actor.Name).
		SetSender(service.owner.EmailAddress).
		AddTo(follower.Actor.EmailAddress)

	// Send the confirmation email
	err := service.serverEmail.Send(
		service.smtp,
		message,
		"follower-confirmation",
		mapof.Any{
			// Parent info available to the template
			"Actor": actor,

			// Follower info available to the template
			"FollowerID": follower.FollowerID.Hex(),
			"Email":      follower.Actor.EmailAddress,
			"Name":       follower.Actor.Name,
			"Secret":     follower.Data.GetString("secret"),

			// Domain info available to the template
			"Owner": service.owner,
			"Host":  service.host(),
			"Label": service.label,
		},
	)

	if err != nil {
		return derp.Wrap(err, "service.DomainEmail.SendFollowConfirmation", "Error sending follow confirmation email to user", follower.Actor.EmailAddress)
	}

	return nil
}

func (service *DomainEmail) SendFollowerActivity(follower model.Follower, activity mapof.Any) error {

	// Build the email message
	message := mail.NewMSG().
		SetSubject("New Activity from "+activity.GetString("actor.name")).
		SetSender(service.owner.EmailAddress).
		AddTo(follower.Actor.EmailAddress).
		AddHeader("List-Unsubscribe", follower.UnsubscribeLink(service.host()))

	host := service.host()

	// Send the activity email
	err := service.serverEmail.Send(
		service.smtp,
		message,
		"follower-activity",
		mapof.Any{

			// Parent info available to the template
			"ParentLink": follower.ParentURL(host),

			// Follower info available to the template
			"FollowerID": follower.FollowerID.Hex(),
			"Email":      follower.Actor.EmailAddress,
			"Name":       follower.Actor.Name,
			"Secret":     follower.Data.GetString("secret"),

			// Activity info available to the template
			"Activity": activity,

			// Domain info available to the template
			"Owner": service.owner,
			"Host":  host,
			"Label": service.label,
		},
	)

	if err != nil {
		return derp.Wrap(err, "service.DomainEmail.SendFollowerEmail", "Error sending follower email to user", follower.Actor.EmailAddress)
	}

	return nil
}

/******************************************
 * Helper Methods
 ******************************************/

func (service *DomainEmail) host() string {
	return domain.Protocol(service.hostname) + service.hostname
}
