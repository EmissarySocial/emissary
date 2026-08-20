package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestAttachmentSchema confirms that every Attachment property round-trips through its schema
func TestAttachmentSchema(t *testing.T) {

	attachment := NewAttachment("TEMP", primitive.NewObjectID())
	s := schema.New(AttachmentSchema())

	table := []tableTestItem{
		{"attachmentId", "123456781234567812345678", nil},
		{"objectId", "876543218765432187654321", nil},
		{"objectType", "Stream", nil},
		{"original", "ORIGINAL", nil},
		{"contentType", "image/png", nil},
		{"category", "CATEGORY", nil},
		{"label", "LABEL", nil},
		{"description", "DESCRIPTION", nil},
		{"url", "http://example.com", nil},
		{"status", "READY", nil},
		{"height", "100", 100},
		{"width", "200", 200},
		{"duration", "100", 100},
		{"rank", "1", 1},
	}

	tableTest_Schema(t, &s, &attachment, table)
}

// TestAttachmentSchema_Nested confirms that the embedded AttachmentRules are reachable by path
func TestAttachmentSchema_Nested(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeStream, primitive.NewObjectID())
	s := schema.New(AttachmentSchema())

	require.Nil(t, s.Set(&attachment, "rules.width", 640))
	require.Nil(t, s.Set(&attachment, "rules.height", 480))
	require.Equal(t, 640, attachment.Rules.Width)
	require.Equal(t, 480, attachment.Rules.Height)

	value, err := s.Get(&attachment, "rules.width")
	require.Nil(t, err)
	require.Equal(t, 640, value)
}

// TestAttachmentSchema_Enums confirms that the object type and status are restricted
func TestAttachmentSchema_Enums(t *testing.T) {

	s := schema.New(AttachmentSchema())

	for _, objectType := range []string{AttachmentObjectTypeDomain, AttachmentObjectTypeSearchTag, AttachmentObjectTypeStream, AttachmentObjectTypeUser} {
		attachment := NewEmptyAttachment()
		require.Nil(t, s.Set(&attachment, "objectType", objectType), "objectType=%q", objectType)
		require.Equal(t, objectType, attachment.ObjectType)
	}

	for _, status := range []string{AttachmentStatusReady, AttachmentStatusWorking} {
		attachment := NewEmptyAttachment()
		require.Nil(t, s.Set(&attachment, "status", status), "status=%q", status)
		require.Equal(t, status, attachment.Status)
	}

	// RULE: Values outside the enum must never reach the record.  Note that the enum
	// is case-sensitive, so "stream" and "USER" are as invalid as "Nonsense".
	for _, objectType := range []string{"Nonsense", "stream", "USER"} {
		attachment := NewEmptyAttachment()
		require.NotNil(t, s.Set(&attachment, "objectType", objectType), "objectType=%q must be rejected", objectType)
		require.Equal(t, "", attachment.ObjectType)
	}

	for _, status := range []string{"ready", "DONE", "Ready"} {
		attachment := NewEmptyAttachment()
		require.NotNil(t, s.Set(&attachment, "status", status), "status=%q must be rejected", status)
		require.Equal(t, "", attachment.Status)
	}

	// An empty string is the "unset" value, and is accepted by both enums
	for _, property := range []string{"objectType", "status"} {
		attachment := NewEmptyAttachment()
		require.Nil(t, s.Set(&attachment, property, ""), "property=%q", property)
	}

	// Surrounding whitespace is trimmed before the enum is checked, so a padded value
	// is accepted and stored in its trimmed form.
	{
		attachment := NewEmptyAttachment()
		require.Nil(t, s.Set(&attachment, "objectType", "  Stream  "))
		require.Equal(t, AttachmentObjectTypeStream, attachment.ObjectType)
	}

	// Markup is stripped *before* the enum is checked, so a value made entirely of
	// tags collapses to "" and is accepted as unset rather than reported as invalid.
	// Nothing is stored either way, but the caller gets no error to react to.
	{
		attachment := NewEmptyAttachment()
		require.Nil(t, s.Set(&attachment, "objectType", "<script>"))
		require.Equal(t, "", attachment.ObjectType)
	}
}

// TestAttachmentSchema_Bounds confirms that oversized and malformed values are handled
func TestAttachmentSchema_Bounds(t *testing.T) {

	s := schema.New(AttachmentSchema())

	// RULE: Every string field is bounded, so a caller cannot inflate a record.
	// Over-long values are *truncated*, not rejected -- Set returns no error, and the
	// stored value is silently clipped to the limit.
	table := []struct {
		property string
		maxLen   int
	}{
		{"category", 64},
		{"label", 64},
		{"description", 1024},
		{"original", 1024},
		{"contentType", 255},
	}

	for _, row := range table {

		attachment := NewEmptyAttachment()

		require.Nil(t, s.Set(&attachment, row.property, repeatRune('x', row.maxLen)), "%s at max length", row.property)

		stored, err := s.Get(&attachment, row.property)
		require.Nil(t, err)
		require.Equal(t, row.maxLen, len(stored.(string)), "%s at max length", row.property)

		require.Nil(t, s.Set(&attachment, row.property, repeatRune('x', row.maxLen*2)), "%s over max length", row.property)

		stored, err = s.Get(&attachment, row.property)
		require.Nil(t, err)
		require.Equal(t, row.maxLen, len(stored.(string)), "%s must be truncated to its limit", row.property)
	}

	// RULE: The URL must be a real URL, so it cannot smuggle a javascript: payload
	for _, value := range []string{"https://example.com/photo.png", "http://example.com", ""} {
		attachment := NewEmptyAttachment()
		require.Nil(t, s.Set(&attachment, "url", value), "url=%q", value)
	}

	for _, value := range []string{"javascript:alert(1)", "not a url at all", "://missing-scheme"} {
		attachment := NewEmptyAttachment()
		require.NotNil(t, s.Set(&attachment, "url", value), "url=%q must be rejected", value)
		require.Equal(t, "", attachment.URL)
	}

	// RULE: ObjectIDs must be valid hex, of the correct length.  Unlike the enums,
	// an empty string is refused here too.
	for _, value := range []string{"", "not-hex", "1234", "123456781234567812345678zz"} {
		attachment := NewEmptyAttachment()
		require.NotNil(t, s.Set(&attachment, "attachmentId", value), "attachmentId=%q must be rejected", value)
		require.True(t, attachment.AttachmentID.IsZero())
	}
}

/******************************************
 * Getter / Setter Interfaces
 ******************************************/

// TestAttachment_GetPointer confirms that every writable field hands back a usable pointer
func TestAttachment_GetPointer(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeStream, primitive.NewObjectID())

	// Every string field, written through its pointer
	for _, name := range []string{"objectType", "category", "label", "description", "url", "original", "contentType", "status"} {

		pointer, ok := attachment.GetPointer(name)
		require.True(t, ok, "property=%q", name)

		stringPointer, ok := pointer.(*string)
		require.True(t, ok, "property=%q must be a *string", name)

		*stringPointer = "VALUE-" + name
	}

	require.Equal(t, "VALUE-objectType", attachment.ObjectType)
	require.Equal(t, "VALUE-category", attachment.Category)
	require.Equal(t, "VALUE-label", attachment.Label)
	require.Equal(t, "VALUE-description", attachment.Description)
	require.Equal(t, "VALUE-url", attachment.URL)
	require.Equal(t, "VALUE-original", attachment.Original)
	require.Equal(t, "VALUE-contentType", attachment.ContentType)
	require.Equal(t, "VALUE-status", attachment.Status)

	// Every integer field, written through its pointer
	for index, name := range []string{"rank", "height", "width", "duration"} {

		pointer, ok := attachment.GetPointer(name)
		require.True(t, ok, "property=%q", name)

		intPointer, ok := pointer.(*int)
		require.True(t, ok, "property=%q must be an *int", name)

		*intPointer = index + 1
	}

	require.Equal(t, 1, attachment.Rank)
	require.Equal(t, 2, attachment.Height)
	require.Equal(t, 3, attachment.Width)
	require.Equal(t, 4, attachment.Duration)

	// The nested rules come back as a pointer to the real struct, not a copy
	{
		pointer, ok := attachment.GetPointer("rules")
		require.True(t, ok)

		rulesPointer, ok := pointer.(*AttachmentRules)
		require.True(t, ok)

		rulesPointer.Width = 999
		require.Equal(t, 999, attachment.Rules.Width)
	}

	// RULE: The two ObjectID fields are deliberately not pointer-writable, because
	// they are stored as ObjectIDs and written through SetString instead.
	for _, name := range []string{"attachmentId", "objectId", "", "unknown-property", "URL", "Label"} {
		pointer, ok := attachment.GetPointer(name)
		require.False(t, ok, "property=%q must not be writable", name)
		require.Equal(t, "", pointer, "the miss value is an empty string, not nil")
	}
}

// TestAttachment_GetStringOK confirms that the ObjectID fields are readable as hex strings
func TestAttachment_GetStringOK(t *testing.T) {

	attachment := NewAttachment(AttachmentObjectTypeStream, primitive.NewObjectID())

	value, ok := attachment.GetStringOK("attachmentId")
	require.True(t, ok)
	require.Equal(t, attachment.AttachmentID.Hex(), value)

	value, ok = attachment.GetStringOK("objectId")
	require.True(t, ok)
	require.Equal(t, attachment.ObjectID.Hex(), value)

	// An empty Attachment still reports zeroed hex, never an empty string
	empty := NewEmptyAttachment()
	value, ok = empty.GetStringOK("objectId")
	require.True(t, ok)
	require.Equal(t, "000000000000000000000000", value)

	for _, name := range []string{"", "label", "unknown-property", "AttachmentID"} {
		value, ok := attachment.GetStringOK(name)
		require.False(t, ok, "property=%q", name)
		require.Equal(t, "", value)
	}
}

// TestAttachment_SetString confirms that only valid ObjectIDs are accepted
func TestAttachment_SetString(t *testing.T) {

	attachment := NewEmptyAttachment()

	require.True(t, attachment.SetString("attachmentId", "123456781234567812345678"))
	require.Equal(t, "123456781234567812345678", attachment.AttachmentID.Hex())

	require.True(t, attachment.SetString("objectId", "876543218765432187654321"))
	require.Equal(t, "876543218765432187654321", attachment.ObjectID.Hex())

	// RULE: A malformed ID is refused, and leaves the previous value intact
	for _, value := range []string{"", "not-hex", "1234", "123456781234567812345678ab", "zzzzzzzzzzzzzzzzzzzzzzzz"} {
		require.False(t, attachment.SetString("attachmentId", value), "value=%q must be rejected", value)
		require.Equal(t, "123456781234567812345678", attachment.AttachmentID.Hex(), "value=%q must not have been written", value)
	}

	// Unknown properties are refused outright
	for _, name := range []string{"", "label", "url", "unknown-property"} {
		require.False(t, attachment.SetString(name, "VALUE"), "property=%q", name)
	}
}

// TestAttachmentRulesSchema confirms that AttachmentRules round-trip through their schema
func TestAttachmentRulesSchema(t *testing.T) {

	rules := NewAttachmentRules()
	s := schema.New(AttachmentRulesSchema())

	require.Nil(t, s.Set(&rules, "width", 640))
	require.Nil(t, s.Set(&rules, "height", 480))
	require.Nil(t, s.Set(&rules, "extensions.0", "webp"))
	require.Nil(t, s.Set(&rules, "extensions.1", "png"))

	require.Equal(t, 640, rules.Width)
	require.Equal(t, 480, rules.Height)
	require.Equal(t, sliceof.String{"webp", "png"}, rules.Extensions)

	value, err := s.Get(&rules, "extensions.0")
	require.Nil(t, err)
	require.Equal(t, "webp", value)

	// RULE: The bitrate is intentionally absent from the schema, so it can only be
	// set in code (by a Template), never by an inbound form post.
	require.NotNil(t, s.Set(&rules, "bitrate", 128))
	require.Equal(t, 0, rules.Bitrate)
}

// TestAttachmentRules_GetPointer confirms which rule fields are writable
func TestAttachmentRules_GetPointer(t *testing.T) {

	rules := NewAttachmentRules()

	{
		pointer, ok := rules.GetPointer("extensions")
		require.True(t, ok)

		extensions, ok := pointer.(*sliceof.String)
		require.True(t, ok)

		*extensions = sliceof.String{"webp"}
		require.Equal(t, sliceof.String{"webp"}, rules.Extensions)
	}

	for index, name := range []string{"height", "width"} {

		pointer, ok := rules.GetPointer(name)
		require.True(t, ok, "property=%q", name)

		intPointer, ok := pointer.(*int)
		require.True(t, ok, "property=%q", name)

		*intPointer = index + 1
	}

	require.Equal(t, 1, rules.Height)
	require.Equal(t, 2, rules.Width)

	// RULE: The bitrate is not writable through the schema.  Unlike Attachment,
	// these misses return nil rather than an empty string.
	for _, name := range []string{"bitrate", "", "unknown-property", "Width"} {
		pointer, ok := rules.GetPointer(name)
		require.False(t, ok, "property=%q must not be writable", name)
		require.Nil(t, pointer)
	}
}

// repeatRune returns a string of the requested length, for testing length limits
func repeatRune(value rune, count int) string {
	result := make([]rune, count)
	for index := range result {
		result[index] = value
	}
	return string(result)
}
