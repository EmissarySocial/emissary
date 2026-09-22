package model

import (
	"encoding/json"
	"testing"

	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Only a WritableDomain can be saved, or used where an AccessLister is required
var _ data.Object = &WritableDomain{}
var _ AccessLister = &WritableDomain{}

// newWritableDomainFixture returns a WritableDomain with a value in every field a document stores,
// including the journal, so a round trip has something to lose.
func newWritableDomainFixture() WritableDomain {

	domain := NewWritableDomain()
	domain.Domain = newDomainCloneFixture()
	domain.Hostname = "example.com"
	domain.IconID = primitive.NewObjectID()
	domain.PrivateKey = "-----BEGIN PRIVATE KEY-----"
	domain.DatabaseVersion = 35
	domain.CreateDate = 1700000000000
	domain.UpdateDate = 1700000001000
	domain.Note = "Stored"
	domain.Revision = 7

	return domain
}

// A read-only Domain cannot be saved: it lacks the journal half of data.Object.
func TestDomain_IsNotADataObject(t *testing.T) {

	_, ok := any(&Domain{}).(data.Object)
	require.False(t, ok, "a *Domain must not satisfy data.Object")

	_, ok = any(&Domain{}).(AccessLister)
	require.False(t, ok, "a *Domain must not satisfy AccessLister")
}

// The stored document keeps the shape it had when the journal was embedded in Domain: every field
// at the top level, with no "domain" or "journal" subdocument.
func TestWritableDomain_StoresAFlatDocument(t *testing.T) {

	encoded, err := bson.Marshal(newWritableDomainFixture())
	require.NoError(t, err)

	document := bson.M{}
	require.NoError(t, bson.Unmarshal(encoded, &document))

	require.NotContains(t, document, "domain")
	require.NotContains(t, document, "journal")
	require.Contains(t, document, "_id")
	require.Contains(t, document, "label")
	require.Contains(t, document, "privateKey")
	require.Contains(t, document, "createDate")
	require.Contains(t, document, "signature")
	require.Equal(t, "example.com", document["hostname"])
	require.Equal(t, int64(7), document["signature"])
}

// A document in the stored shape decodes into a WritableDomain and encodes back to the same document.
func TestWritableDomain_RoundTripsTheStoredDocument(t *testing.T) {

	original := newWritableDomainFixture()

	encoded, err := bson.Marshal(original)
	require.NoError(t, err)

	decoded := NewWritableDomain()
	require.NoError(t, bson.Unmarshal(encoded, &decoded))
	require.Equal(t, original, decoded)

	reencoded, err := bson.Marshal(decoded)
	require.NoError(t, err)
	require.Equal(t, encoded, reencoded)
}

// The journal never leaves the server: JSON carries the Domain's fields flat and nothing from the journal.
func TestWritableDomain_JSONOmitsTheJournal(t *testing.T) {

	encoded, err := json.Marshal(newWritableDomainFixture())
	require.NoError(t, err)

	document := mapof.NewAny()
	require.NoError(t, json.Unmarshal(encoded, &document))

	require.NotContains(t, document, "domain")
	require.NotContains(t, document, "journal")
	require.NotContains(t, document, "createDate")
	require.NotContains(t, document, "signature")
	require.NotContains(t, document, "privateKey")
	require.Equal(t, "example.com", document["hostname"])
}

// The constructor starts from NewDomain, so every map and slice is allocated and the journal is blank.
func TestNewWritableDomain(t *testing.T) {

	domain := NewWritableDomain()

	require.Equal(t, NewDomain(), domain.Domain)
	require.True(t, domain.IsNew())
}

// The schema setters write through to the embedded Domain
func TestWritableDomain_SetString(t *testing.T) {

	domain := NewWritableDomain()
	iconID := primitive.NewObjectID()

	require.True(t, domain.SetString("iconId", iconID.Hex()))
	require.Equal(t, iconID, domain.Domain.IconID)

	require.True(t, domain.SetString("iconId", ""))
	require.True(t, domain.Domain.IconID.IsZero())

	require.True(t, domain.SetString("mlsGroupIds", "a,b"))
	require.Equal(t, []string{"a", "b"}, []string(domain.Domain.MLSGroupIDs))

	// Virtual fields accept a write without storing it
	require.True(t, domain.SetString("iconUrl", "https://example.com"))
	require.True(t, domain.SetString("imageUrl", "https://example.com"))

	require.False(t, domain.SetString("iconId", "not-an-object-id"))
	require.False(t, domain.SetString("no-such-field", "value"))
}

// The pointer getter reaches the embedded Domain's fields, so a write through it lands on the record
func TestWritableDomain_GetPointer(t *testing.T) {

	domain := NewWritableDomain()

	pointer, ok := domain.GetPointer("label")
	require.True(t, ok)

	label, ok := pointer.(*string)
	require.True(t, ok)
	*label = "Written"
	require.Equal(t, "Written", domain.Domain.Label)

	_, ok = domain.GetPointer("no-such-field")
	require.False(t, ok)
}
