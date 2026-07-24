package service

import (
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome"
	"github.com/benpate/exp"
)

// signinLockoutWindow is the rolling window over which failed signin attempts are
// counted. The lockout is temporary by construction: once no new failures land
// inside the window, the account signs in normally again. There is no permanent,
// third-party-triggerable lock to weaponize, and a failed login never mutates the
// victim's stored password.
const signinLockoutWindow = 15 * time.Minute

// signinLockoutThreshold is the number of failed attempts -- per username, across
// ALL source IP addresses -- within signinLockoutWindow that locks the account.
// Counting across IPs is deliberate: the username is the only key a distributed,
// IP-rotating attacker cannot escape, so it is the layer that catches exactly the
// attack a per-IP block (the Digital Dome) cannot.
const signinLockoutThreshold = 10

// SterankoSigninService implements the steranko.SigninService interface.  This tells
// steranko how to procees users who have too many failed signin attempts
type SterankoSigninService struct {
	userService *User
	digitalDome func() *dome.Dome // resolved lazily: the Dome lives in the server factory, so calling it at construction time would require the full factory graph
	clientIP    func(*http.Request) string
	session     data.Session
}

// NewSterankoSigninService returns a SterankoSigninService bound to the provided
// factory and database session.
func NewSterankoSigninService(factory *Factory, session data.Session) SterankoSigninService {
	return SterankoSigninService{
		userService: factory.User(),
		digitalDome: factory.DigitalDome, // method value -- not invoked until a signin path needs it
		clientIP:    factory.ClientIP,
		session:     session,
	}
}

// SigninSuccess removes failed signin attempts for the provided username.
func (s SterankoSigninService) SigninSuccess(request *http.Request, username string) {

	if err := s.ClearSigninAttempts(username); err != nil {
		derp.Report(err)
	}
}

// SigninFailure logs a new failed signin attempt for the provided username. It is
// called by steranko ONLY on a genuine bad-credential attempt (not when an already
// locked account is refused), so every call here is real abuse: it counts toward
// the username's lockout window AND penalizes the source IP in the Digital Dome.
func (s SterankoSigninService) SigninFailure(request *http.Request, username string) {

	// Record the attempt so it counts toward this username's rolling lockout window.
	signinAttempt := model.NewSigninAttempt(username, s.clientIPOf(request), userAgentOf(request))

	if err := s.collection().Save(&signinAttempt, ""); err != nil {
		derp.Report(derp.Wrap(err, "SterankoSigninService.SigninFailure", "Saving signin attempt", signinAttempt))
	}

	// Penalize the source IP: a single guessing IP bans itself in the Digital Dome
	// after enough failures, no matter which username(s) it targets.
	s.blockIP(request)

	// Drop this username's attempts that have aged out of the window, so the
	// collection stays bounded even for usernames that never sign in successfully.
	s.pruneExpiredAttempts(username)
}

// IsSigninLocked returns TRUE if the provided username has reached the failed-attempt
// threshold within the rolling lockout window.
func (s SterankoSigninService) IsSigninLocked(request *http.Request, username string) bool {

	const location = "SterankoSigninService.IsSigninLocked"

	// Count only failures recent enough to still be inside the window; older
	// attempts age out, which is what makes the lockout temporary.
	cutoff := time.Now().Add(-signinLockoutWindow).UnixMilli()
	failureCount, err := s.collection().Count(exp.Equal("username", username).And(exp.GreaterThan("createDate", cutoff)))

	// RULE: fail closed. If we cannot read the attempt history, deny the signin
	// rather than fall open to unlimited guessing.
	if err != nil {
		derp.Report(derp.Wrap(err, location, "Counting recent signin attempts for user", username))
		return true
	}

	// Under the threshold, the account is not locked.
	if failureCount < signinLockoutThreshold {
		return false
	}

	// The account IS locked. Penalize the requesting IP: hammering a locked account
	// is abuse too, and this is what bans the individual IPs of a distributed attack
	// while the username-level lock holds. steranko does not report locked-out
	// attempts to SigninFailure (that would renew the lock forever), so this is the
	// only place the locked-path penalty can be applied.
	s.blockIP(request)

	// On the transition INTO the locked state, notify the account owner once. The
	// stored password is never changed -- see User.NotifySigninLockout.
	if failureCount == signinLockoutThreshold {
		s.userService.NotifySigninLockout(s.session, username)
	}

	return true
}

// ClearSigninAttempts removes every recorded signin attempt for the provided username.
func (s SterankoSigninService) ClearSigninAttempts(username string) error {

	if err := s.collection().HardDelete(exp.Equal("username", username)); err != nil {
		return derp.Wrap(err, "SterankoSigninService.ClearSigninAttempts", "Clearing signin attempts for user", username)
	}

	return nil
}

// pruneExpiredAttempts hard-deletes the provided username's signin attempts that
// have aged out of the lockout window. Errors are reported but not returned: this
// is opportunistic cleanup on the failure path and must not mask the signin result.
func (s SterankoSigninService) pruneExpiredAttempts(username string) {

	cutoff := time.Now().Add(-signinLockoutWindow).UnixMilli()

	if err := s.collection().HardDelete(exp.Equal("username", username).And(exp.LessThan("createDate", cutoff))); err != nil {
		derp.Report(derp.Wrap(err, "SterankoSigninService.pruneExpiredAttempts", "Pruning expired signin attempts", username))
	}
}

// blockIP records one abuse event against the request's client IP in the Digital
// Dome. It tolerates a nil request (the steranko interface always supplies one, but
// callers and tests may not).
func (s SterankoSigninService) blockIP(request *http.Request) {

	if request != nil {
		s.digitalDome().Block(request)
	}
}

// clientIPOf resolves the request's client IP using the configured trusted-proxy
// strategy, returning "" for a nil request.
func (s SterankoSigninService) clientIPOf(request *http.Request) string {

	if request == nil {
		return ""
	}

	return s.clientIP(request)
}

// userAgentOf returns the request's User-Agent header, or "" for a nil request.
func userAgentOf(request *http.Request) string {

	if request == nil {
		return ""
	}

	return request.UserAgent()
}

func (s SterankoSigninService) collection() data.Collection {
	return s.session.Collection("SigninAttempt")
}
