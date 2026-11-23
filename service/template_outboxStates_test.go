package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestInvalidOutboxStates_StreamTemplate confirms that a Stream Template may set the flag on any of its States,
// under each model name that builds a Stream
func TestInvalidOutboxStates_StreamTemplate(t *testing.T) {

	for _, modelName := range []string{"Stream", "None", ""} {
		t.Run("model "+modelName, func(t *testing.T) {
			template := model.NewTemplate("test-article", nil)
			template.Model = modelName
			template.States = mapof.Object[model.State]{
				"default":   {StateID: "default"},
				"published": {StateID: "published", IsOutbox: true},
				"featured":  {StateID: "featured", IsOutbox: true},
			}

			require.Empty(t, invalidOutboxStates(&template))
		})
	}
}

// TestInvalidOutboxStates_NoFlag confirms that a Template that sets no flag has nothing to report, whatever it builds
func TestInvalidOutboxStates_NoFlag(t *testing.T) {

	for _, modelName := range []string{"", "Stream", "Outbox", "User", "Domain"} {
		t.Run("model "+modelName, func(t *testing.T) {
			template := model.NewTemplate("test", nil)
			template.Model = modelName
			template.States = mapof.Object[model.State]{"default": {StateID: "default"}, "LIVE": {StateID: "LIVE"}}

			require.Empty(t, invalidOutboxStates(&template))
		})
	}
}

// TestInvalidOutboxStates_NoStates confirms that a Template with no States at all reports nothing here
func TestInvalidOutboxStates_NoStates(t *testing.T) {

	template := model.NewTemplate("test-outbox", nil)
	template.Model = "Outbox"

	require.Empty(t, invalidOutboxStates(&template))
}

// TestInvalidOutboxStates_OtherModels confirms that the flag is refused on every Template that does not build a Stream
func TestInvalidOutboxStates_OtherModels(t *testing.T) {

	for _, modelName := range []string{"Outbox", "User", "Domain", "Group"} {
		t.Run("model "+modelName, func(t *testing.T) {
			template := model.NewTemplate("test-other", nil)
			template.Model = modelName
			template.States = mapof.Object[model.State]{
				"default": {StateID: "default"},
				"LIVE":    {StateID: "LIVE", IsOutbox: true},
			}

			errs := invalidOutboxStates(&template)
			require.Len(t, errs, 1)
			require.Equal(t, "Only a Stream Template can set 'outbox' on a State", errs[0].Message)
			require.Contains(t, errs[0].Details, any("template: test-other"))
			require.Contains(t, errs[0].Details, any("model: "+modelName))
			require.Contains(t, errs[0].Details, any("state: LIVE"))
		})
	}
}

// TestInvalidOutboxStates_ReportsEveryState confirms that each flagged State gets its own error, in StateID order
func TestInvalidOutboxStates_ReportsEveryState(t *testing.T) {

	template := model.NewTemplate("test-outbox", nil)
	template.Model = "Outbox"
	template.States = mapof.Object[model.State]{
		"zulu":    {StateID: "zulu", IsOutbox: true},
		"default": {StateID: "default"},
		"alpha":   {StateID: "alpha", IsOutbox: true},
		"mike":    {StateID: "mike", IsOutbox: true},
	}

	errs := invalidOutboxStates(&template)
	require.Len(t, errs, 3)
	require.Contains(t, errs[0].Details, any("state: alpha"))
	require.Contains(t, errs[1].Details, any("state: mike"))
	require.Contains(t, errs[2].Details, any("state: zulu"))
}

// TestValidateTemplates_ReportsOutboxStates confirms that validateTemplates reports the flag on an outbox Template,
// and reports nothing for the same flag on a Stream Template
func TestValidateTemplates_ReportsOutboxStates(t *testing.T) {

	const message = "Only a Stream Template can set 'outbox' on a State"

	templates := newDefaultChildTemplates()

	outbox := templates["test-outbox"]
	outbox.States = mapof.Object[model.State]{"default": {StateID: "default", IsOutbox: true}}
	templates["test-outbox"] = outbox

	collection := templates["test-collection"]
	collection.States = mapof.Object[model.State]{"default": {StateID: "default", IsOutbox: true}}
	templates["test-collection"] = collection

	service := &Template{templatePrep: templates}
	reported := make([]string, 0)

	for _, err := range service.validateTemplates() {
		if err.Message == message {
			require.Contains(t, err.Details, any("template: test-outbox"))
			reported = append(reported, err.Message)
		}
	}

	require.Len(t, reported, 1, "validateTemplates must report the outbox Template, and only it")
}
