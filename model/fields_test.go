package model

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// bsonNames returns the bson name of every field on a struct, following inlined embeds.
func bsonNames(value any) []string {
	return bsonNamesOfType(reflect.TypeOf(value))
}

// bsonNamesOfType returns the bson name of every field on a struct type, following inlined embeds.
func bsonNamesOfType(structType reflect.Type) []string {

	result := make([]string, 0, structType.NumField())

	for index := range structType.NumField() {

		field := structType.Field(index)
		name, options, _ := strings.Cut(field.Tag.Get("bson"), ",")

		// An inlined struct (journal.Journal, etc) contributes its fields to the parent document
		if strings.Contains(options, "inline") {
			if field.Type.Kind() == reflect.Struct {
				result = append(result, bsonNamesOfType(field.Type)...)
			}
			continue
		}

		if name != "" && name != "-" {
			result = append(result, name)
		}
	}

	return result
}

// TestFieldProjections guards every Mongo projection in this package. Fields() names are plain
// strings that no compiler checks: a name that does not match a bson tag asks Mongo for a field
// that cannot exist, and the field it was MEANT to name silently loads as its zero value.
//
// That failure is invisible -- no error, no panic, just an empty struct member -- and it is how
// "places" outlived the rename to "location", and "activityPubUsername" outlived "webfingerUsername".
//
// This test only checks that projected names EXIST. It deliberately does not require that every
// bson field be projected: trimming is the entire point of a summary type, and the journal fields
// are excluded on purpose.
func TestFieldProjections(t *testing.T) {

	table := []struct {
		name   string
		target any
		fields []string
	}{
		{"Annotation", Annotation{}, AnnotationFields()},
		{"Circle", Circle{}, CircleFields()},
		{"Collection", Collection{}, Collection{}.Fields()},
		{"FollowerSummary", FollowerSummary{}, FollowerSummaryFields()},
		{"FollowingSummary", FollowingSummary{}, FollowingSummaryFields()},
		{"Group", Group{}, GroupFields()},
		{"Identity", Identity{}, Identity{}.Fields()},
		{"MerchantAccount", MerchantAccount{}, MerchantAccount{}.Fields()},
		{"NewsItem", NewsItem{}, NewsItemFields()},
		{"Notification", Notification{}, NotificationFields()},
		{"ObjectLink", ObjectLink{}, ObjectLink{}.Fields()},
		{"OutboxMessage", OutboxMessage{}, OutboxMessageFields()},
		{"OutboxMessageSummary", OutboxMessageSummary{}, OutboxMessageSummaryFields()},
		{"Privilege", Privilege{}, Privilege{}.Fields()},
		{"Response", Response{}, Response{}.Fields()},
		{"Rule", Rule{}, Rule{}.Fields()},
		{"RuleSummary", RuleSummary{}, RuleSummaryFields()},
		{"SearchResult", SearchResult{}, SearchResult{}.Fields()},
		{"SearchTag", SearchTag{}, SearchTag{}.Fields()},
		{"StreamSummary", StreamSummary{}, StreamSummaryFields()},
		{"UserSummary", UserSummary{}, UserSummaryFields()},
		{"Webhook", Webhook{}, WebhookFields()},
	}

	for _, item := range table {
		t.Run(item.name, func(t *testing.T) {
			require.Subset(t, bsonNames(item.target), item.fields, "projected a name with no matching bson field")
		})
	}
}

// TestFieldProjections_Renames pins the specific names that outlived a rename, so that a revert
// fails loudly instead of silently blanking a field again.
func TestFieldProjections_Renames(t *testing.T) {

	streamSummary := StreamSummaryFields()
	require.Contains(t, streamSummary, "location")
	require.NotContains(t, streamSummary, "places", "renamed to location")

	identity := Identity{}.Fields()
	require.Contains(t, identity, "webfingerUsername")
	require.NotContains(t, identity, "activityPubUsername", "renamed to webfingerUsername")
}

// TestResponseFields_Computed guards the fields that Response's own methods read. A Response can
// load and look complete while ActivityPubURL returns a URL missing its actor prefix, or
// IsMyself/RolesToGroupIDs answer against a zero UserID.
func TestResponseFields_Computed(t *testing.T) {

	fields := Response{}.Fields()

	require.Contains(t, fields, "actor", "ActivityPubURL and Toot build URLs from Actor")
	require.Contains(t, fields, "userId", "the AccessLister methods key off UserID")
	require.Contains(t, fields, "createDate", "liked-list paginates on CreateDate")
}
