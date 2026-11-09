package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	mockdb "github.com/benpate/data-mock"
	"github.com/stretchr/testify/require"
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
