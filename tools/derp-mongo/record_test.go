package derpmongo

import (
	"errors"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewRecord(t *testing.T) {

	inner := derp.NotFound("service.Inner.Load", "record not found")
	err := derp.Wrap(inner, "service.Outer.Do", "could not do the thing")

	record := newRecord(err, derp.ErrorCode(err))

	assert.Equal(t, 404, record.StatusCode)
	assert.Equal(t, "service.Inner.Load", record.Location, "the record stores the ROOT location")
	assert.Equal(t, "record not found", record.Message, "the record stores the ROOT message")
	assert.Equal(t, err, record.Error)
	assert.Equal(t, StatusNew, record.Status)
	assert.NotEqual(t, primitive.NilObjectID, record.RecordID)
	assert.NotZero(t, record.CreateDate)
}

func TestNewRecord_SignatureAgreesWithSignatureOf(t *testing.T) {

	// These two must never disagree.  The plugin writes one; anything reading a live error
	// computes the other, and a drift between them would split one defect into two items.
	errs := []error{
		derp.Wrap(derp.NotFound("service.Inner.Load", "gone"), "service.Outer.Do", "failed"),
		derp.Internal("service.A", "lookup one.example.com: no such host"),
		derp.Validation("email address is invalid"),
		errors.New("boom"),
		nil,
	}

	for _, err := range errs {
		record := newRecord(err, derp.ErrorCode(err))
		assert.Equal(t, SignatureOf(err), record.Signature, "signing %v", err)
	}
}

func TestNewRecord_UndecidedByDefault(t *testing.T) {

	record := newRecord(derp.Internal("service.A", "broken"), 500)

	assert.Equal(t, StatusNew, record.Status)
	assert.Empty(t, record.StatusNote, "nothing has been decided yet")
	assert.Zero(t, record.StatusDate)
}

func TestNewRecord_HandlesNil(t *testing.T) {

	assert.NotPanics(t, func() {
		record := newRecord(nil, 0)
		assert.Equal(t, StatusNew, record.Status)
		assert.Equal(t, SignatureOf(nil), record.Signature)
	})
}

func TestNewRecord_SeparatesPlainErrors(t *testing.T) {

	first := newRecord(errors.New("boom"), 500)
	second := newRecord(errors.New("totally different message"), 500)

	assert.NotEqual(t, first.Signature, second.Signature)
}

func TestRecord_BSONFieldNames(t *testing.T) {

	// Triage reads this document through a struct of its own, so the field NAMES are the
	// contract between the two packages.
	record := newRecord(derp.Internal("service.A", "broken"), 500)

	raw, err := bson.Marshal(record)
	require.NoError(t, err)

	decoded := bson.M{}
	require.NoError(t, bson.Unmarshal(raw, &decoded))

	assert.Equal(t, record.Signature, decoded["signature"])
	assert.Equal(t, string(StatusNew), decoded["status"])
	assert.Contains(t, decoded, "statusCode")
	assert.Contains(t, decoded, "createDate")

	assert.NotContains(t, decoded, "statusNote", "an undecided record carries no note")
	assert.NotContains(t, decoded, "statusDate", "an undecided record carries no decision date")
}

func TestRecord_BSONRoundTrip(t *testing.T) {

	record := newRecord(derp.Internal("service.A", "broken"), 500)
	record.Status = StatusIgnored
	record.StatusNote = "the far end is broken, not us"
	record.StatusDate = primitive.NewDateTimeFromTime(record.CreateDate.Time())

	raw, err := bson.Marshal(record)
	require.NoError(t, err)

	decoded := bson.M{}
	require.NoError(t, bson.Unmarshal(raw, &decoded))

	assert.Equal(t, string(StatusIgnored), decoded["status"])
	assert.Equal(t, "the far end is broken, not us", decoded["statusNote"])
	assert.Contains(t, decoded, "statusDate")
}

func TestStatusValues(t *testing.T) {

	// Triage matches on these strings, so they are a stored contract and not just labels
	assert.Equal(t, "new", string(StatusNew))
	assert.Equal(t, "fixed", string(StatusFixed))
	assert.Equal(t, "ignored", string(StatusIgnored))
}
