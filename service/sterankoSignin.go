package service

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
)

// SterankoSigninService implements the steranko.SigninService interface.  This tells
// steranko how to procees users who have too many failed signin attempts
type SterankoSigninService struct {
	userService *User
	session     data.Session
}

func NewSterankoSigninService(factory *Factory, session data.Session) SterankoSigninService {
	return SterankoSigninService{
		userService: factory.User(),
		session:     session,
	}
}

// SigninSuccess removes failed signin attempts for the provided username.
func (s SterankoSigninService) SigninSuccess(request *http.Request, username string) {

	if err := s.ClearSigninAttempts(username); err != nil {
		derp.Report(err)
	}
}

// SigninFailure logs a new failed signin attempt for the provided username.
func (s SterankoSigninService) SigninFailure(request *http.Request, username string) {
	signinAttempt := model.NewSigninAttempt(username)

	if err := s.collection().Save(&signinAttempt, ""); err != nil {
		derp.Report(derp.Wrap(err, "SterankoSigninService.SigninFailure", "Unable to save signin attempt", signinAttempt))
	}
}

// IsSigninLocked returns TRUE if the provided username has more than 5 failed signin attempts.
func (s SterankoSigninService) IsSigninLocked(request *http.Request, username string) bool {

	failureCount, err := s.collection().Count(exp.Equal("username", username))

	if err != nil {
		derp.Report(derp.Wrap(err, "SterankoSigninService.IsSigninLocked", "Unable to count signin attempts for user", username))
		return true
	}

	if failureCount >= 5 {

		// Send lockout email every 5 attempts to notify the user that their account is being targeted.
		if (failureCount % 5) == 0 {
			s.userService.Lockout(s.session, username)
		}

		return true
	}

	return false
}

func (s SterankoSigninService) ClearSigninAttempts(username string) error {

	if err := s.collection().HardDelete(exp.Equal("username", username)); err != nil {
		return derp.Wrap(err, "SterankoSigninService.ClearSigninAttempts", "Unable to clear signin attempts for user", username)
	}

	return nil
}

func (s SterankoSigninService) collection() data.Collection {
	return s.session.Collection("SigninAttempt")
}
