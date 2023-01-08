package gofed

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/moby/locker"
)

// Database implements the go-fed Database interface.  Individual methods include
// comments imported from: https://go-fed.org/ref/activity/pub#The-Database-Interface
//
// The pub.Database interface is how you provide stateful information to the go-fed
// library. It gives your app the flexibility to use whatever implementation you wish
// to use or are already using.
//
// This interface follows several principles:
//
//  1. The database is IRI-centric. It treats *url.URL parameters as a primary key.
//  2. The Lock and Unlock methods will be appropriately called for a particular IRI.
//  3. It does not handle concepts like transactions out of the box. However, since
//     it follows the context.Context pattern, it is possible for your particular
//     implementation to still use concepts like transactions.
//  4. It does not handle concepts like caching out of the box. However, some APIs
//     are designed to work with pages of a Collection, rather than the entire
//     Collection, to facilitate implementations that want to develop their own
//     caching capability.
type Database struct {
	userService   *service.User
	inboxService  *service.Inbox
	outboxService *service.Outbox
	hostname      string

	locks *locker.Locker
}

func NewDatabase(userService *service.User, inboxService *service.Inbox, outboxService *service.Outbox, hostname string) Database {
	return Database{
		userService:   userService,
		inboxService:  inboxService,
		outboxService: outboxService,
		hostname:      hostname,

		locks: locker.New(),
	}
}
