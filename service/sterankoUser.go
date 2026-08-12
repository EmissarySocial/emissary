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
			return derp.Wrap(err, location, "Loading user")
		}

		// If the User has moved to a new server, then they cannot sign in
		if user.MovedTo != "" {
			return derp.Forbidden(location, "User moved to new server", user.MovedTo)
		}

		return nil
	}

	return derp.Internal(location, "Invalid result provided.  This should never happen")
}

// LoadBySubject retrieves a single User by the "sub" claim in their session
// token, which Emissary populates with the User's hex-encoded UserID (see
// claims). Keying on the immutable UserID means revalidation keeps working even
// after a User changes their username or email.
func (service SterankoUserService) LoadBySubject(subject string, result steranko.User) error {

	const location = "service.SterankoUserService.LoadBySubject"

	// Confirm that we have been sent a User pointer
	user, isUser := result.(*model.User)

	if !isUser {
		return derp.Internal(location, "Invalid result provided.  This should never happen")
	}

	// The subject is the User's hex-encoded ObjectID
	userID, err := primitive.ObjectIDFromHex(subject)

	if err != nil {
		return derp.Wrap(err, location, "Invalid subject (expected hex ObjectID)", subject)
	}

	// Load the user from the database
	if err := service.userService.LoadByID(service.session, userID, user); err != nil {
		return derp.Wrap(err, location, "Loading user")
	}

	// RULE: If the User has moved to a new server, then their session is no longer valid
	if user.MovedTo != "" {
		return derp.Forbidden(location, "User moved to new server", user.MovedTo)
	}

	return nil
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

// RequestPasswordReset generates a new password reset code and emails it to the User.
func (service SterankoUserService) RequestPasswordReset(user steranko.User) error {

	const location = "service.SterankoUserService.RequestPasswordReset"

	// RULE: Steranko Users must be model.User objects
	modelUser, ok := user.(*model.User)

	if !ok {
		return derp.Internal(location, "Steranko User is not a valid object.  This should never happen", user)
	}

	// RULE: Reset codes are single-use, so mint a new code in case a previous one was used or expired
	if err := service.userService.MakeNewPasswordResetCode(service.session, modelUser, model.PasswordResetDurationReset); err != nil {
		return derp.Wrap(err, location, "Making password reset code")
	}

	// Send the password reset email
	return service.domainEmail.SendPasswordReset(modelUser)
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

	// Create a new claim for the target user
	claims, err := service.claims(user)

	if err != nil {
		return nil, derp.Wrap(err, location, "Creating JWT claims for masquerade")
	}

	// Mark this claim as a "masquerade" claim, which allows the administrator to act as the target user
	claims.Masquerade = true

	// Success.
	return &claims, nil
}

// Claims creates a new JWT claim object for the provided User. This implements the Steranko UserService interface.
func (service SterankoUserService) Claims(user steranko.User) (jwt.Claims, error) {

	const location = "service.SterankoUserService.Claims"

	claims, err := service.claims(user)
	if err != nil {
		return nil, derp.Wrap(err, location, "Creating JWT claims")
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
			return model.Authorization{}, derp.Wrap(err, location, "Loading Identity for User")
		}
	}

	identityID := iif(identity.IsNew(), primitive.NilObjectID, identity.IdentityID)

	// Claims returns all access privileges given to this user.  A part of the "steranko.User" interface.
	result := model.Authorization{
		UserID:      user.UserID,
		IdentityID:  identityID,
		GroupIDs:    user.GroupIDs,
		DomainOwner: user.IsOwner,
		Revalidate:  time.Now().Unix(), // Marks this as a full User session that Steranko re-verifies once it ages past the revalidation window (see GetRevalidationTime)
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.UserID.Hex(),                                // Stable identifier Steranko uses to re-load the User during revalidation (see LoadBySubject)
			IssuedAt:  jwt.NewNumericDate(time.Now()),                   // Current create date.  (Used by Steranko to refresh tokens)
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(10, 0, 0)), // Expires ten years from now (but re-validated sooner by Steranko)
		},
	}

	return result, nil
}

// Close is required to implement the steranko.UserService interface
func (service SterankoUserService) Close() {

}
