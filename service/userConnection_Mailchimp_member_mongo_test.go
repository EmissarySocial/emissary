package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// insertReadyMailchimpConnection writes a READY connection whose stored API key ciphertext is known
func insertReadyMailchimpConnection(t *testing.T, server data.Server, ciphertext string) model.UserConnection {

	t.Helper()

	userConnection := model.NewUserConnection()
	userConnection.UserID = primitive.NewObjectID()
	userConnection.Type = model.UserConnectionTypeMailchimp
	userConnection.IsActive.Set(true)
	userConnection.Status = model.UserConnectionStatusReady
	userConnection.Vault.Encrypted = mapof.String{model.UserConnectionVaultAPIKey: ciphertext}

	session, err := server.Session(context.Background())
	require.NoError(t, err)
	defer session.Close()

	require.NoError(t, session.Collection("UserConnection").Save(&userConnection, "test setup"))

	return userConnection
}

// loadStoredStatus returns the status a connection has in the database
func loadStoredStatus(t *testing.T, server data.Server, userConnectionID primitive.ObjectID) string {

	t.Helper()

	session, err := server.Session(context.Background())
	require.NoError(t, err)
	defer session.Close()

	result := model.NewUserConnection()
	require.NoError(t, session.Collection("UserConnection").Load(exp.Equal("_id", userConnectionID), &result))

	return result.Status
}

// reportInTransaction runs mailchimp_reportMemberError the way a consumer does, inside a
// transaction that commits only when the returned error is nil
func reportInTransaction(t *testing.T, server data.Server, userConnection *model.UserConnection, mailchimpErr error) error {

	t.Helper()

	service := UserConnection{}

	var reported error

	_, _ = server.WithTransaction(context.Background(), func(session data.Session) (any, error) {
		reported = service.mailchimp_reportMemberError(session, userConnection, mailchimpErr)
		return nil, reported
	})

	return reported
}

// TestMailchimpReportMemberError_RejectedCredentialSurvivesTheTransaction pins the flag that a
// rollback used to erase
func TestMailchimpReportMemberError_RejectedCredentialSurvivesTheTransaction(t *testing.T) {

	server, _ := newReplicaSetSession(t)
	userConnection := insertReadyMailchimpConnection(t, server, "ciphertext-A")

	for _, statusCode := range []int{http.StatusUnauthorized, http.StatusForbidden} {

		rejected := derp.Internal("test", "Mailchimp refused this key", derp.WithCode(statusCode))

		require.NoError(t, reportInTransaction(t, server, &userConnection, rejected), "a refused key is final, so the task ends")
		require.Equal(t, model.UserConnectionStatusReconnect, loadStoredStatus(t, server, userConnection.UserConnectionID))
	}
}

// TestMailchimpReportMemberError_ReplacedCredentialIsNotFlagged confirms a key changed elsewhere
// is not marked as refused
func TestMailchimpReportMemberError_ReplacedCredentialIsNotFlagged(t *testing.T) {

	server, _ := newReplicaSetSession(t)
	stored := insertReadyMailchimpConnection(t, server, "ciphertext-B")

	// This task loaded the connection before the User replaced its key on another server
	stale := stored
	stale.Vault.Encrypted = mapof.String{model.UserConnectionVaultAPIKey: "ciphertext-A"}

	rejected := derp.Internal("test", "Mailchimp refused this key", derp.WithCode(http.StatusUnauthorized))

	require.NoError(t, reportInTransaction(t, server, &stale, rejected))
	require.Equal(t, model.UserConnectionStatusReady, loadStoredStatus(t, server, stored.UserConnectionID))
}

// TestMailchimpReportMemberError_TransientErrorsAreRetried confirms anything but a refused key
// is returned unchanged and leaves the connection alone
func TestMailchimpReportMemberError_TransientErrorsAreRetried(t *testing.T) {

	server, _ := newReplicaSetSession(t)
	userConnection := insertReadyMailchimpConnection(t, server, "ciphertext-A")

	for _, statusCode := range []int{http.StatusBadRequest, http.StatusTooManyRequests, http.StatusBadGateway} {

		transient := derp.Internal("test", "Mailchimp failed", derp.WithCode(statusCode))

		require.Equal(t, transient, reportInTransaction(t, server, &userConnection, transient))
		require.Equal(t, model.UserConnectionStatusReady, loadStoredStatus(t, server, userConnection.UserConnectionID))
	}
}
