package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SterankoUserService is a wrapper/adapter that makes the User service compatable with Steranko.
type SterankoUserService struct {
	identityService *Identity
	userService     *User
	domainEmail     *DomainEmail
	session         data.Session
}

// NewSterankoUserService returns a fully populated SterankoUserService.
func NewSterankoUserService(identityService *Identity, userService *User, domainEmail *DomainEmail, session data.Session) SterankoUserService {
	return SterankoUserService{
		identityService: identityService,
		userService:     userService,
		domainEmail:     domainEmail,
		session:         session,
	}
}

// New creates a newly initialized User that is ready to use
func (service SterankoUserService) New() steranko.User {
	result := model.NewUser()
	return &result
}

// Load retrieves a single User from the database
func (service SterankoUserService) Load(username string, result steranko.User) error {

	const location = "service.SterankoUserService.Load"

	// Confirm that we have been sent a User pointer
	if user, isUser := result.(*model.User); isUser {

		// Load the user from the database
		if err := service.userService.LoadByUsernameOrEmail(service.session, username, user); err != nil {
			return derp.Wrap(err, location, "Unable to load user")
		}

		// If the User has moved to a new server, then they cannot sign in
		if user.MovedTo != "" {
			return derp.Forbidden(location, "User moved to new server", user.MovedTo)
		}

		return nil
	}

	return derp.Internal(location, "Invalid result provided.  This should never happen")
}

// Save inserts/updates a single User in the database
func (service SterankoUserService) Save(user steranko.User, comment string) error {

	const location = "service.SterankoUserService.Save"

	if user, ok := user.(*model.User); ok {
		return service.userService.Save(service.session, user, comment)
	}

	return derp.Internal(location, "Steranko User is not a valid object.  This should never happen", user)
}

// Delete removes a single User from the database
func (service SterankoUserService) Delete(user steranko.User, comment string) error {

	const location = "service.SterankoUserService.Delete"

	if user, ok := user.(*model.User); ok {
		return service.userService.Delete(service.session, user, comment)
	}

	return derp.Internal(location, "Steranko User is not a valid object.  This should never happen", user)
}

// RequestPasswordReset is not currently implemented in this service. (TODO)
func (service SterankoUserService) RequestPasswordReset(user steranko.User) error {

	const location = "service.SterankoUserService.RequestPasswordReset"

	if user, ok := user.(*model.User); ok {
		return service.domainEmail.SendPasswordReset(user)
	}

	return derp.Internal(location, "Steranko User is not a valid object.  This should never happen", user)
}

// NewClaims creates a new JWT claim object
func (service SterankoUserService) NewClaims() jwt.Claims {
	result := model.NewAuthorization()
	return &result
}

// MasqueradeAs creates a new JWT claim object for the provided User, and sets the "Masquerade" flag to TRUE
func (service SterankoUserService) MasqueradeAs(user *model.User) (jwt.Claims, error) {

	const location = "service.SterankoUserService.MasqueradeAs"

	// If the User has moved to a new server, then they cannot be masqueraded
	if user.MovedTo != "" {
		return nil, derp.Forbidden(location, "User moved to new server", user.MovedTo)
	}

	claims, err := service.claims(user)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to create JWT claims for masquerade")
	}
	claims.Masquerade = true

	return &claims, nil
}

// Claims creates a new JWT claim object for the provided User. This implements the Steranko UserService interface.
func (service SterankoUserService) Claims(user steranko.User) (jwt.Claims, error) {

	const location = "service.SterankoUserService.Claims"

	claims, err := service.claims(user)
	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to create JWT claims")
	}
	return &claims, nil
}

// claims is a common method used to create claims for users and administrators
func (service SterankoUserService) claims(sterankoUser steranko.User) (model.Authorization, error) {

	const location = "service.SterankoUserService.Claims"

	user, isCorrectType := sterankoUser.(*model.User)

	if !isCorrectType {
		return model.Authorization{}, derp.Internal(location, "Steranko User is not a valid object.  This should never happen", user)
	}

	// Look up the Identity for this User.  If missing, NBD..
	identity := model.NewIdentity()
	if err := service.identityService.LoadByEmailAddress(service.session, user.EmailAddress, &identity); err != nil {
		if !derp.IsNotFound(err) {
			return model.Authorization{}, derp.Wrap(err, location, "Unable to load Identity for User")
		}
	}

	identityID := iif(identity.IsNew(), primitive.NilObjectID, identity.IdentityID)

	// Claims returns all access privileges given to this user.  A part of the "steranko.User" interface.
	result := model.Authorization{
		UserID:      user.UserID,
		IdentityID:  identityID,
		GroupIDs:    user.GroupIDs,
		DomainOwner: user.IsOwner,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),                   // Current create date.  (Used by Steranko to refresh tokens)
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(10, 0, 0)), // Expires ten years from now (but re-validated sooner by Steranko)
		},
	}

	return result, nil
}

// Close is required to implement the steranko.UserService interface
func (service SterankoUserService) Close() {

}
