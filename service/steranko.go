package service

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// SterankoUserService is a wrapper/adapter that makes the User service compatable with Steranko.
type SterankoUserService struct {
	userService User
}

// New creates a newly initialized User that is ready to use
func (service SterankoUserService) New() steranko.User {
	return service.userService.New()
}

// Load retrieves a single User from the database
func (service SterankoUserService) Load(username string) (steranko.User, error) {
	return service.userService.LoadByUsername(username)
}

// Save inserts/updates a single User in the database
func (service SterankoUserService) Save(user steranko.User, comment string) error {

	if user, ok := user.(*model.User); ok {
		return service.userService.Save(user, comment)
	}

	return derp.New(derp.CodeInternalError, "ghost.service.SterankoUserService.Save", "Steranko User is not a valid object.  This should never happen", user)
}

// Delete removes a single User from the database
func (service SterankoUserService) Delete(user steranko.User, comment string) error {

	if user, ok := user.(*model.User); ok {
		return service.userService.Delete(user, comment)
	}

	return derp.New(derp.CodeInternalError, "ghost.service.SterankoUserService.Delete", "Steranko User is not a valid object.  This should never happen", user)
}

// Close cleans up any connections opened by the service.
func (service SterankoUserService) Close() {
	service.userService.Close()
}

// RequestPasswordReset is not currently implemented in this service. (TODO)
func (service SterankoUserService) RequestPasswordReset(user steranko.User) error {
	return nil
}
