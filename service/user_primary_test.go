package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestApplyPrimaryUser_OnlyAnOwner verifies that a User who is not an owner cannot stay primary,
// and that neither this case nor an owner who is not primary reads the database
func TestApplyPrimaryUser_OnlyAnOwner(t *testing.T) {

	// A zero service has no collection, so reaching the database would panic
	service := &User{}

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	// RULE: The flag is cleared from a User who is not an owner, including a demoted owner
	demoted := model.NewUser()
	demoted.IsPrimary = true
	require.NoError(t, service.applyPrimaryUser(session, &demoted))
	require.False(t, demoted.IsPrimary)

	// An owner who is not primary is left as they are
	owner := model.NewUser()
	owner.IsOwner = true
	require.NoError(t, service.applyPrimaryUser(session, &owner))
	require.True(t, owner.IsOwner)
	require.False(t, owner.IsPrimary)

	// So is a User who is neither
	member := model.NewUser()
	require.NoError(t, service.applyPrimaryUser(session, &member))
	require.False(t, member.IsOwner)
	require.False(t, member.IsPrimary)
}

// newPrimaryUserTest returns a User service whose Domain record is stored in a fresh replica-set database
func newPrimaryUserTest(t *testing.T) (*User, *Domain, data.Session) {

	t.Helper()
	domainService, session := newPrimaryUserDomain(t)

	return &User{domainService: domainService}, domainService, session
}

// newTestPrimaryUser returns an owner flagged primary
func newTestPrimaryUser() model.User {

	user := model.NewUser()
	user.IsOwner = true
	user.IsPrimary = true

	return user
}

// TestUser_RecordPrimaryUser verifies that the Domain names the primary user through every change
func TestUser_RecordPrimaryUser(t *testing.T) {

	service, domainService, session := newPrimaryUserTest(t)
	users := session.Collection("User")

	// Saving another User on a domain with no primary user changes nothing
	member := model.NewUser()
	require.NoError(t, users.Save(&member, "test"))
	require.NoError(t, service.recordPrimaryUser(session, &member))
	requireStoredPrimaryUser(t, domainService, session, primitive.NilObjectID)

	// Saving a primary user names them
	first := newTestPrimaryUser()
	require.NoError(t, users.Save(&first, "test"))
	require.NoError(t, service.recordPrimaryUser(session, &first))
	requireStoredPrimaryUser(t, domainService, session, first.UserID)

	// Saving another User keeps them
	require.NoError(t, service.recordPrimaryUser(session, &member))
	requireStoredPrimaryUser(t, domainService, session, first.UserID)

	// Moving the flag, in the order applyPrimaryUser saves: the old one is cleared before the new one is written
	second := newTestPrimaryUser()
	first.IsPrimary = false
	require.NoError(t, users.Save(&first, "test"))
	require.NoError(t, service.recordPrimaryUser(session, &first))
	requireStoredPrimaryUser(t, domainService, session, primitive.NilObjectID)
	require.NoError(t, users.Save(&second, "test"))
	require.NoError(t, service.recordPrimaryUser(session, &second))
	requireStoredPrimaryUser(t, domainService, session, second.UserID)

	// Moving it back in the other order ends the same way, because clearing the old one finds the new one named
	first.IsPrimary = true
	second.IsPrimary = false
	require.NoError(t, users.Save(&second, "test"))
	require.NoError(t, users.Save(&first, "test"))
	require.NoError(t, service.recordPrimaryUser(session, &first))
	require.NoError(t, service.recordPrimaryUser(session, &second))
	requireStoredPrimaryUser(t, domainService, session, first.UserID)

	// Deleting the primary user clears it
	require.NoError(t, users.Delete(&first, "test"))
	require.NoError(t, service.recordPrimaryUser(session, &first))
	requireStoredPrimaryUser(t, domainService, session, primitive.NilObjectID)

	// So does clearing the only primary user's flag
	second.IsPrimary = true
	require.NoError(t, users.Save(&second, "test"))
	require.NoError(t, service.recordPrimaryUser(session, &second))
	requireStoredPrimaryUser(t, domainService, session, second.UserID)

	second.IsPrimary = false
	require.NoError(t, users.Save(&second, "test"))
	require.NoError(t, service.recordPrimaryUser(session, &second))
	requireStoredPrimaryUser(t, domainService, session, primitive.NilObjectID)
}

// TestUser_RecordPrimaryUser_DemotedOwner verifies that a primary flag left on a non-owner never names them
func TestUser_RecordPrimaryUser_DemotedOwner(t *testing.T) {

	service, domainService, session := newPrimaryUserTest(t)

	demoted := newTestPrimaryUser()
	demoted.IsOwner = false

	require.NoError(t, service.recordPrimaryUser(session, &demoted))
	requireStoredPrimaryUser(t, domainService, session, primitive.NilObjectID)
}

// TestUser_RecordPrimaryUser_ReadsNothing verifies that saving anyone but the named primary user costs no query
func TestUser_RecordPrimaryUser_ReadsNothing(t *testing.T) {

	// A nil session panics on any read, so these pass only if nothing is read
	t.Run("WithoutAPrimaryUser", func(t *testing.T) {
		service := &User{domainService: &Domain{}}
		member := model.NewUser()
		require.NoError(t, service.recordPrimaryUser(nil, &member))
	})

	t.Run("ForAnotherUser", func(t *testing.T) {
		domainService := &Domain{}
		writableDomain := model.NewWritableDomain()
		writableDomain.PrimaryUserID = primitive.NewObjectID()
		domainService.publish(writableDomain)

		service := &User{domainService: domainService}
		member := model.NewUser()
		require.NoError(t, service.recordPrimaryUser(nil, &member))
	})

	t.Run("ForTheNamedPrimaryUser", func(t *testing.T) {
		primary := newTestPrimaryUser()
		domainService := &Domain{}
		writableDomain := model.NewWritableDomain()
		writableDomain.PrimaryUserID = primary.UserID
		domainService.publish(writableDomain)

		service := &User{domainService: domainService}
		require.NoError(t, service.recordPrimaryUser(nil, &primary))
	})
}
