package service

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// invalidDefaultChildrenMessages runs invalidDefaultChildren on the named Template in a prep area
// built from newDefaultChildTemplates, and returns each error's message
func invalidDefaultChildrenMessages(t *testing.T, templateID string, entries ...mapof.Any) []string {

	t.Helper()

	service := &Template{templatePrep: newDefaultChildTemplates(entries...)}
	template, exists := service.templatePrep[templateID]
	require.True(t, exists, "template %q is not in the prep area", templateID)

	result := make([]string, 0)

	for _, err := range service.invalidDefaultChildren(&template) {
		result = append(result, err.Message)
	}

	return result
}

// TestInvalidDefaultChildren_NoChildren confirms that a Template with no default children has nothing to report
func TestInvalidDefaultChildren_NoChildren(t *testing.T) {
	require.Empty(t, invalidDefaultChildrenMessages(t, "test-outbox"))
	require.Empty(t, invalidDefaultChildrenMessages(t, "test-collection"))
}

// TestInvalidDefaultChildren_ValidEntries confirms that entries that User.Save can create pass
func TestInvalidDefaultChildren_ValidEntries(t *testing.T) {
	require.Empty(t, invalidDefaultChildrenMessages(t, "test-outbox", articlesEntry(), notesEntry()))
}

// TestInvalidDefaultChildren_OnlyAnOutbox confirms that default children on any other Template are refused,
// because nothing would ever create them
func TestInvalidDefaultChildren_OnlyAnOutbox(t *testing.T) {

	service := &Template{templatePrep: newDefaultChildTemplates()}
	collection := service.templatePrep["test-collection"]
	collection.DefaultChildren = []mapof.Any{articlesEntry(), notesEntry()}

	errs := service.invalidDefaultChildren(&collection)

	// One error for the Template, not one for each entry
	require.Len(t, errs, 1)
	require.Equal(t, "Only an outbox Template can declare 'defaultChildren'", errs[0].Message)
}

// TestInvalidDefaultChildren_UndefinedTemplate confirms that an entry naming no Template is refused
func TestInvalidDefaultChildren_UndefinedTemplate(t *testing.T) {

	for _, templateID := range []string{"no-such-template", ""} {
		entry := articlesEntry()
		entry["templateId"] = templateID

		require.Equal(t,
			[]string{"Default child names a Template that is not defined"},
			invalidDefaultChildrenMessages(t, "test-outbox", entry),
			"templateId %q", templateID,
		)
	}
}

// TestInvalidDefaultChildren_TemplateOutsideTheOutbox confirms that a Template an outbox cannot hold is refused
func TestInvalidDefaultChildren_TemplateOutsideTheOutbox(t *testing.T) {
	require.Equal(t,
		[]string{"Default child names a Template that an outbox cannot hold"},
		invalidDefaultChildrenMessages(t, "test-outbox", mapof.Any{"templateId": "test-page", "label": "Page"}),
	)
}

// TestInvalidDefaultChildren_RefusedValue confirms that a value the child Template's schema refuses is caught
// at load, including a "data" value that only the child Template declares
func TestInvalidDefaultChildren_RefusedValue(t *testing.T) {

	for _, values := range []mapof.Any{
		{"data.layout": "notess"},
		{"lable": "Articles"},
		{"groups.viewer.0": "anonymous"},
	} {
		entry := articlesEntry()

		for key, value := range values {
			entry[key] = value
		}

		require.Equal(t,
			[]string{"Default child sets a value that its Template's schema refuses"},
			invalidDefaultChildrenMessages(t, "test-outbox", entry),
			"values %v", values,
		)
	}
}

// TestInvalidDefaultChildren_ReportsEveryBadEntry confirms that one bad entry does not hide the next one
func TestInvalidDefaultChildren_ReportsEveryBadEntry(t *testing.T) {

	misspelled := articlesEntry()
	misspelled["lable"] = "Articles"

	require.Equal(t,
		[]string{
			"Default child names a Template that is not defined",
			"Default child names a Template that an outbox cannot hold",
			"Default child sets a value that its Template's schema refuses",
		},
		invalidDefaultChildrenMessages(t, "test-outbox",
			mapof.Any{"templateId": "no-such-template"},
			articlesEntry(),
			mapof.Any{"templateId": "test-page", "label": "Page"},
			misspelled,
			notesEntry(),
		),
	)
}

// TestValidateTemplates_ReportsDefaultChildren confirms that validateTemplates reports a bad entry, which is
// what keeps a broken outbox from being published and failing every new User's sign-up instead
func TestValidateTemplates_ReportsDefaultChildren(t *testing.T) {

	service := &Template{templatePrep: newDefaultChildTemplates(mapof.Any{"templateId": "no-such-template"})}

	found := false

	for _, err := range service.validateTemplates() {
		if err.Message == "Default child names a Template that is not defined" {
			require.Contains(t, err.Details, any("template: test-outbox"))
			found = true
		}
	}

	require.True(t, found, "validateTemplates must report the bad entry")
}

// TestInvalidDefaultChildren_ErrorsNameTheEntry confirms that each error names the outbox, the entry's position,
// and the Template it named, so that an author can find the line to fix
func TestInvalidDefaultChildren_ErrorsNameTheEntry(t *testing.T) {

	service := &Template{templatePrep: newDefaultChildTemplates(articlesEntry(), mapof.Any{"templateId": "no-such-template"})}
	outbox := service.templatePrep["test-outbox"]

	errs := service.invalidDefaultChildren(&outbox)
	require.Len(t, errs, 1)
	require.Equal(t, []any{"template: test-outbox", "index: 1", "templateId: no-such-template"}, errs[0].Details)
	require.True(t, derp.IsValidationError(errs[0]))
}
