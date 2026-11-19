package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestKeyPackage_GetJSONLD verifies the document a KeyPackage is served as, including that it is
// addressed to Public by the full IRI
func TestKeyPackage_GetJSONLD(t *testing.T) {

	service := KeyPackage{host: "https://this.example"}

	keyPackage := model.NewKeyPackage()
	keyPackage.KeyPackageID = primitive.NewObjectID()
	keyPackage.UserID = primitive.NewObjectID()
	keyPackage.MediaType = "message/mls"
	keyPackage.Encoding = "base64"
	keyPackage.Content = "AAEC"
	keyPackage.GeneratorID = "https://client.example"
	keyPackage.GeneratorName = "Client"
	keyPackage.Ciphersuite = "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"

	result := service.GetJSONLD(&keyPackage)
	userURL := "https://this.example/@" + keyPackage.UserID.Hex()

	// O6: a bare "Public" is not public on GoToSocial, Pixelfed, PeerTube, Lemmy 0.19 or Fedify
	require.Equal(t, vocab.NamespaceActivityStreamsPublic, result[vocab.PropertyTo])

	require.Equal(t, userURL+"/pub/keyPackages/"+keyPackage.KeyPackageID.Hex(), result[vocab.PropertyID])
	require.Equal(t, userURL, result[vocab.PropertyAttributedTo])
	require.Equal(t, "message/mls", result[vocab.PropertyMediaType])
	require.Equal(t, "base64", result[vocab.PropertyEncoding])
	require.Equal(t, "AAEC", result[vocab.PropertyContent])
	require.Equal(t, keyPackage.Ciphersuite, result[vocab.PropertyMLSCiphersuite])
	require.Equal(t, "A binary-encoded cryptographic key", result[vocab.PropertySummary], "a record with no summary gets the generic one")

	// A client-authored summary is kept
	keyPackage.Summary = "🔑🌵🎈"
	require.Equal(t, "🔑🌵🎈", service.GetJSONLD(&keyPackage)[vocab.PropertySummary])
}
