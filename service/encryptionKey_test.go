package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// testPrivatePEM stands in for a private key that must never reach an error
const testPrivatePEM = "-----BEGIN RSA PRIVATE KEY-----private-pem-s3cr3t-----END RSA PRIVATE KEY-----"

// TestEncryptionKey_OmitsPrivateKeyFromErrors requires that no failure reports the private key
func TestEncryptionKey_OmitsPrivateKeyFromErrors(t *testing.T) {

	// BUG-173: PrivatePEM is json:"privatePEM", so these errors printed the private key to the
	// console as well as storing it in ErrorLog.
	service := EncryptionKey{}

	t.Run("Save", func(t *testing.T) {
		encryptionKey := newTestEncryptionKey()
		err := service.Save(brokenSession{}, &encryptionKey, "test")

		require.Equal(t, "service.EncryptionKey.Save", derp.Location(err))
		secretcheck.RequireAbsent(t, err, testPrivatePEM)
	})

	t.Run("Delete", func(t *testing.T) {
		encryptionKey := newTestEncryptionKey()
		err := service.Delete(brokenSession{}, &encryptionKey, "test")

		require.Equal(t, "service.EncryptionKey.Delete", derp.Location(err))
		secretcheck.RequireAbsent(t, err, testPrivatePEM)
	})

	t.Run("DeleteByParentID", func(t *testing.T) {
		encryptionKey := newTestEncryptionKey()
		session := brokenSession{records: map[string][]data.Object{"EncryptionKey": {&encryptionKey}}}
		err := service.DeleteByParentID(session, encryptionKey.ParentID, "test")

		require.Equal(t, "service.EncryptionKey.DeleteByParentID", derp.Location(err))
		require.Equal(t, "Deleting key", derp.Message(err))
		secretcheck.RequireAbsent(t, err, testPrivatePEM)
	})
}

// newTestEncryptionKey returns a key carrying testPrivatePEM
func newTestEncryptionKey() model.EncryptionKey {
	encryptionKey := model.NewEncryptionKey()
	encryptionKey.ParentID = primitive.NewObjectID()
	encryptionKey.PrivatePEM = testPrivatePEM
	return encryptionKey
}
