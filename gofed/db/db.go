package db

import (
	"sync"

	"github.com/EmissarySocial/emissary/service"
)

type Database struct {
	factory       Factory
	userService   *service.User
	inboxService  *service.Inbox
	outboxService *service.Outbox

	// Enables mutations. A sync.Mutex per ActivityPub ID.
	locks *sync.Map

	// The host domain of our service, for detecting ownership.
	hostname string
}

func NewDatabase(factory Factory, userService *service.User, inboxService *service.Inbox, outboxService *service.Outbox, hostname string) *Database {
	return &Database{
		factory:       factory,
		userService:   userService,
		inboxService:  inboxService,
		outboxService: outboxService,
		locks:         &sync.Map{},
		hostname:      hostname,
	}
}
