package service

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
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

func (service *DomainEmail) SendWelcome(user *model.User) {

	data := maps.Map{
		"User":     user,
		"Owner":    service.owner,
		"Hostname": service.hostname,
		"Label":    service.label,
	}

	err := service.serverEmail.Send(
		service.smtp,
		"user-welcome",
		service.owner.EmailAddress,
		[]string{user.Username},
		"Welcome to Emissary",
		data,
	)

	derp.Report(err)
}
