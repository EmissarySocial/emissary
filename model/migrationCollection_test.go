package model

import (
	"strings"
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestMigrationCollections verifies that every collection has a distinct key and path in the expected namespaces
func TestMigrationCollections(t *testing.T) {

	keys := map[string]bool{}
	paths := map[string]bool{}

	for _, collection := range MigrationCollections() {
		require.True(t, strings.HasPrefix(collection.Key, "emissary:"), collection.Key)
		require.True(t, strings.HasPrefix(collection.Path, "emissary-"), collection.Path)
		require.False(t, keys[collection.Key], "duplicate key %s", collection.Key)
		require.False(t, paths[collection.Path], "duplicate path %s", collection.Path)
		keys[collection.Key] = true
		paths[collection.Path] = true
	}

	// BUG-83: these two were advertised under names that no export route served
	require.False(t, keys["emissary:conversation"])
	require.True(t, paths["emissary-newsFeed"])
	require.False(t, paths["emissary-newsItem"])
}

// TestUser_MigrationJSONLD verifies that the actor advertises every collection beneath its export URL
func TestUser_MigrationJSONLD(t *testing.T) {

	user := NewUser()
	user.UserID = primitive.NewObjectID()
	user.ProfileURL = "https://example.com/@" + user.UserID.Hex()

	migration := user.MigrationJSONLD()
	exportURL := user.ActivityPubURL() + "/export/"

	require.Len(t, migration, len(MigrationCollections())+4, "the four standard collections, plus each native one")
	require.Equal(t, exportURL+"outbox", migration["outbox"])

	for _, collection := range MigrationCollections() {
		require.Equal(t, exportURL+collection.Path, migration[collection.Key], collection.Key)
	}
}

// TestUser_GetJSONLD_OmitsMigration verifies that the export collections stay out of the document
// that the profile fingerprint is calculated from
func TestUser_GetJSONLD_OmitsMigration(t *testing.T) {

	user := NewUser()
	user.ProfileURL = "https://example.com/@user"

	require.NotContains(t, user.GetJSONLD(), vocab.PropertyMigration)
}
