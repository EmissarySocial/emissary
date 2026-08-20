package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
)

// These tests drive SterankoSigninService against a hand-built in-memory
// data.Collection. benpate/data-mock can't be used here because it matches
// predicate fields against bson tags and won't descend into the `,inline`
// embedded journal.Journal, so it can never match the "createDate" window
// queries this service depends on.

/******************************************
 * signinStore — an in-memory SigninAttempt collection that understands exactly
 * the queries the service issues: Equal("username") combined with a createDate
 * GreaterThan/LessThan range.
 ******************************************/

// signinStore is an in-memory data.Collection of SigninAttempts, used by the tests in this file
type signinStore struct {
	records  []*model.SigninAttempt
	countErr error // when set, Count returns this error (to exercise fail-closed)
}

// Context implements the interface, returning a background context
func (c *signinStore) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *signinStore) Count(criteria exp.Expression, _ ...option.Option) (int64, error) {

	if c.countErr != nil {
		return 0, c.countErr
	}

	var count int64
	for _, record := range c.records {
		if matchesAttempt(criteria, record) {
			count++
		}
	}
	return count, nil
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *signinStore) Save(object data.Object, _ string) error {

	attempt, ok := object.(*model.SigninAttempt)
	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	// Mimic the real data layer, which stamps CreateDate on insert.
	if attempt.CreateDate == 0 {
		attempt.CreateDate = time.Now().UnixMilli()
	}

	stored := *attempt
	c.records = append(c.records, &stored)
	return nil
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *signinStore) HardDelete(criteria exp.Expression) error {

	kept := c.records[:0:0]
	for _, record := range c.records {
		if !matchesAttempt(criteria, record) {
			kept = append(kept, record)
		}
	}
	c.records = kept
	return nil
}

// Load implements the data.Collection interface. Unused by these tests.
func (c *signinStore) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *signinStore) Query(any, exp.Expression, ...option.Option) error { return nil }

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *signinStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

// Delete implements the data.Collection interface, backed by this stub's in-memory records
func (c *signinStore) Delete(data.Object, string) error { return nil }

// matchesAttempt evaluates the service's criteria against a single record. It
// supports Equal on "username" and GreaterThan/LessThan on "createDate"; any
// other field or operator conservatively counts as "no match".
func matchesAttempt(criteria exp.Expression, record *model.SigninAttempt) bool {

	return criteria.Match(func(predicate exp.Predicate) bool {
		switch predicate.Field {

		case "username":
			value, ok := predicate.Value.(string)
			return ok && predicate.Operator == exp.OperatorEqual && record.Username == value

		case "createDate":
			value, ok := predicate.Value.(int64)
			if !ok {
				return false
			}
			switch predicate.Operator {
			case exp.OperatorGreaterThan:
				return record.CreateDate > value
			case exp.OperatorLessThan:
				return record.CreateDate < value
			default:
				return false
			}

		default:
			return false
		}
	})
}

/******************************************
 * notFoundCollection — a data.Collection whose Load always misses, used for the
 * "User" collection so User.LoadByUsername returns NotFound and the lockout
 * notification becomes a safe no-op.
 ******************************************/

// notFoundCollection is a data.Collection whose every read reports NotFound
type notFoundCollection struct{}

// Context implements the interface, returning a background context
func (notFoundCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (notFoundCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, nil
}

// Load implements the data.Collection interface, backed by this stub's in-memory records
func (notFoundCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.NotFound("test", "no such user")
}

// Save implements the data.Collection interface, backed by this stub's in-memory records
func (notFoundCollection) Save(data.Object, string) error { return nil }

// Delete implements the data.Collection interface. Unused by these tests.
func (notFoundCollection) Delete(data.Object, string) error { return nil }

// HardDelete implements the data.Collection interface. Unused by these tests.
func (notFoundCollection) HardDelete(exp.Expression) error { return nil }

// Query implements the data.Collection interface. Unused by these tests.
func (notFoundCollection) Query(any, exp.Expression, ...option.Option) error { return nil }

// Iterator implements the data.Collection interface. Unused by these tests.
func (notFoundCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

/******************************************
 * signinSession — a data.Session that routes "SigninAttempt" to the test store
 * and everything else (i.e. "User") to a NotFound collection.
 ******************************************/

// signinSession is a data.Session that routes "SigninAttempt" to the test store, and everything else to NotFound
type signinSession struct {
	store data.Collection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s signinSession) Collection(name string) data.Collection {
	if name == "SigninAttempt" {
		return s.store
	}
	return notFoundCollection{}
}

// Context implements the interface, returning a background context
func (s signinSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s signinSession) Close() {}

// newSigninService assembles a SterankoSigninService wired to the test store and
// a real (but empty) Dome, bypassing the factory-based constructor.
func newSigninService(store data.Collection) (SterankoSigninService, *dome.Dome) {

	userService := NewUser()
	testDome := dome.New(dome.RemoteAddr)

	service := SterankoSigninService{
		userService: &userService,
		digitalDome: func() *dome.Dome { return testDome },
		clientIP:    dome.RemoteAddr,
		session:     signinSession{store: store},
	}

	return service, testDome
}

// attemptAt builds a stored SigninAttempt for the given username, aged by the
// provided offset from now (a positive age means "in the past").
func attemptAt(username string, age time.Duration) *model.SigninAttempt {
	attempt := model.NewSigninAttempt(username, "1.2.3.4", "test-agent")
	attempt.CreateDate = time.Now().Add(-age).UnixMilli()
	return &attempt
}

// signinRequest builds the POST /signin request that the tests below submit
func signinRequest() *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/signin", nil)
	request.RemoteAddr = "5.6.7.8:9999"
	request.Header.Set("User-Agent", "attacker-agent")
	return request
}

/******************************************
 * Tests
 ******************************************/

// TestSigninLocked_UnderThresholdIsNotLocked verifies that fewer failures than the threshold leave an account unlocked
func TestSigninLocked_UnderThresholdIsNotLocked(t *testing.T) {

	store := &signinStore{}
	for range signinLockoutThreshold - 1 {
		store.records = append(store.records, attemptAt("target@example.com", time.Minute))
	}

	service, _ := newSigninService(store)

	require.False(t, service.IsSigninLocked(signinRequest(), "target@example.com"))
}

// TestSigninLocked_AtThresholdIsLocked verifies that reaching the threshold locks the account and penalizes the source IP
func TestSigninLocked_AtThresholdIsLocked(t *testing.T) {

	store := &signinStore{}
	// Seed ABOVE the threshold so the count != threshold branch is taken and the
	// owner-notification path (which needs a live user) is not exercised here.
	for range signinLockoutThreshold + 2 {
		store.records = append(store.records, attemptAt("target@example.com", time.Minute))
	}

	service, testDome := newSigninService(store)
	t.Cleanup(testDome.Close)

	request := signinRequest()

	// Every locked-out hit penalizes the requesting IP in the Dome; after enough of
	// them the IP crosses the Dome's block threshold (this is what bans the
	// individual IPs of a distributed attack while the username lock holds).
	for range 6 {
		require.True(t, service.IsSigninLocked(request, "target@example.com"))
	}
	require.NotNil(t, testDome.VerifyRequest(request), "repeated locked hits should block the source IP")
}

// TestSigninLocked_OldAttemptsAgeOut verifies that failures older than the lockout window do not count
func TestSigninLocked_OldAttemptsAgeOut(t *testing.T) {

	store := &signinStore{}
	// Many failures, but all older than the window: the account must NOT be locked.
	for range signinLockoutThreshold * 3 {
		store.records = append(store.records, attemptAt("target@example.com", signinLockoutWindow+time.Minute))
	}

	service, _ := newSigninService(store)

	require.False(t, service.IsSigninLocked(signinRequest(), "target@example.com"))
}

// TestSigninLocked_DifferentUsernameIsIsolated verifies that flooding one account does not lock a different one
func TestSigninLocked_DifferentUsernameIsIsolated(t *testing.T) {

	store := &signinStore{}
	for range signinLockoutThreshold + 5 {
		store.records = append(store.records, attemptAt("victim@example.com", time.Minute))
	}

	service, _ := newSigninService(store)

	// A flood against "victim" must not lock a different account.
	require.False(t, service.IsSigninLocked(signinRequest(), "someone-else@example.com"))
}

// TestSigninLocked_CountErrorFailsClosed verifies that a database failure locks the account rather than opening it
func TestSigninLocked_CountErrorFailsClosed(t *testing.T) {

	store := &signinStore{countErr: derp.Internal("test", "database is down")}
	service, _ := newSigninService(store)

	require.True(t, service.IsSigninLocked(signinRequest(), "target@example.com"))
}

// TestSigninFailure_RecordsAttemptWithIPAndAgent verifies that a failed signin records the caller's IP and user agent
func TestSigninFailure_RecordsAttemptWithIPAndAgent(t *testing.T) {

	store := &signinStore{}
	service, _ := newSigninService(store)

	request := signinRequest()
	service.SigninFailure(request, "target@example.com")

	require.Len(t, store.records, 1)
	require.Equal(t, "target@example.com", store.records[0].Username)
	require.Equal(t, "5.6.7.8", store.records[0].IPAddress) // resolved from RemoteAddr
	require.Equal(t, "attacker-agent", store.records[0].UserAgent)
}

// TestSigninFailure_PenalizesSourceIP verifies that repeated failures get the source IP blocked by the Digital Dome
func TestSigninFailure_PenalizesSourceIP(t *testing.T) {

	store := &signinStore{}
	service, testDome := newSigninService(store)
	t.Cleanup(testDome.Close)

	request := signinRequest()

	// Six failures push the source IP over the Dome's block threshold of 5.
	for range 6 {
		service.SigninFailure(request, "target@example.com")
	}

	require.NotNil(t, testDome.VerifyRequest(request), "repeated failures should block the source IP")
}

// TestSigninFailure_PrunesExpiredAttempts verifies that recording a failure also clears attempts older than the window
func TestSigninFailure_PrunesExpiredAttempts(t *testing.T) {

	store := &signinStore{}
	// One recent and one long-expired attempt for the same username.
	store.records = append(store.records, attemptAt("target@example.com", time.Minute))
	store.records = append(store.records, attemptAt("target@example.com", signinLockoutWindow+time.Hour))

	service, testDome := newSigninService(store)
	t.Cleanup(testDome.Close)

	service.SigninFailure(signinRequest(), "target@example.com")

	// The expired record is pruned; the recent one and the new one remain.
	for _, record := range store.records {
		age := time.Now().UnixMilli() - record.CreateDate
		require.Less(t, age, signinLockoutWindow.Milliseconds(), "expired attempts must be pruned")
	}
}

// TestSigninSuccess_ClearsAllAttempts verifies that a successful signin wipes that account's failure history
func TestSigninSuccess_ClearsAllAttempts(t *testing.T) {

	store := &signinStore{}
	store.records = append(store.records, attemptAt("target@example.com", time.Minute))
	store.records = append(store.records, attemptAt("target@example.com", 2*time.Minute))

	service, _ := newSigninService(store)

	service.SigninSuccess(signinRequest(), "target@example.com")

	require.Empty(t, store.records)
}

// TestSigninFailure_ToleratesNilRequest verifies that a nil request does not panic the service
func TestSigninFailure_ToleratesNilRequest(t *testing.T) {

	store := &signinStore{}
	service, testDome := newSigninService(store)
	t.Cleanup(testDome.Close)

	// The steranko interface always supplies a request, but the service must not
	// panic if one is ever nil.
	require.NotPanics(t, func() {
		service.SigninFailure(nil, "target@example.com")
	})
	require.Len(t, store.records, 1)
	require.Equal(t, "", store.records[0].IPAddress)
}
