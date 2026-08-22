package queries

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestUpgradeMongoDB_UpToDateNeedsNoDatabase pins the ordering that makes every routine domain
// start cheap: the version check runs BEFORE the database is touched, so an already-upgraded
// domain costs nothing -- proven here by passing no database at all.
func TestUpgradeMongoDB_UpToDateNeedsNoDatabase(t *testing.T) {

	domain := model.NewDomain()
	domain.DatabaseVersion = 999 // far past the current target

	require.NoError(t, UpgradeMongoDB(context.Background(), nil, &domain))
}

// TestUpgradeMongoDB_PendingUpgradeRequiresDatabase pins the guard on the other side: a domain
// that DOES need upgrades cannot proceed without a connection, and says so instead of panicking.
func TestUpgradeMongoDB_PendingUpgradeRequiresDatabase(t *testing.T) {

	domain := model.NewDomain()
	domain.DatabaseVersion = 0

	require.Error(t, UpgradeMongoDB(context.Background(), nil, &domain))
}
