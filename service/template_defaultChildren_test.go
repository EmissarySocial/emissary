package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
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

// TestInvalidDefaultChildren_StreamTemplate confirms that a Stream Template may declare default children,
// and that each one must be a Template that this Template's role can hold
func TestInvalidDefaultChildren_StreamTemplate(t *testing.T) {

	service := &Template{templatePrep: newDefaultChildTemplates()}
	collection := service.templatePrep["test-collection"]

	collection.DefaultChildren = []mapof.Any{{"templateId": "test-item", "label": "First", "data.size": "small"}}
	require.Empty(t, service.invalidDefaultChildren(&collection))

	// A collection cannot hold another collection, which only an outbox can hold
	collection.DefaultChildren = []mapof.Any{articlesEntry()}
	errs := service.invalidDefaultChildren(&collection)
	require.Len(t, errs, 1)
	require.Equal(t, "Default child names a Template that cannot be placed beneath this one", errs[0].Message)
	require.Contains(t, errs[0].Details, any("containedBy needs: test-collection"))

	// The item's own schema checks the values
	collection.DefaultChildren = []mapof.Any{{"templateId": "test-item", "label": "First", "data.size": "medium"}}
	errs = service.invalidDefaultChildren(&collection)
	require.Len(t, errs, 1)
	require.Equal(t, "Default child sets a value that its Template's schema refuses", errs[0].Message)
}

// TestInvalidDefaultChildren_OnlyAnOutboxOrAStream confirms that default children on a Template that builds
// neither an outbox nor a Stream are refused, because nothing would ever create them
func TestInvalidDefaultChildren_OnlyAnOutboxOrAStream(t *testing.T) {

	service := &Template{templatePrep: newDefaultChildTemplates()}

	for _, modelName := range []string{"Group", "Domain", "User", "Inbox"} {
		template := model.NewTemplate("test-"+modelName, nil)
		template.Model = modelName
		template.TemplateRole = "test-" + modelName
		template.DefaultChildren = []mapof.Any{articlesEntry(), notesEntry()}

		errs := service.invalidDefaultChildren(&template)

		// One error for the Template, not one for each entry
		require.Len(t, errs, 1, modelName)
		require.Equal(t, "Only an outbox or a Stream Template can declare 'defaultChildren'", errs[0].Message, modelName)
	}
}

// TestDefaultChildrenRole confirms where each kind of Template's default children are placed
func TestDefaultChildrenRole(t *testing.T) {

	outbox := model.NewTemplate("test-outbox", nil)
	outbox.Model = "Outbox"
	outbox.TemplateRole = "user-outbox"

	collection := model.NewTemplate("test-collection", nil)
	collection.Model = "Stream"
	collection.TemplateRole = "test-collection"

	unnamedModel := model.NewTemplate("test-unnamed", nil)
	unnamedModel.TemplateRole = "test-unnamed"

	group := model.NewTemplate("test-group", nil)
	group.Model = "Group"
	group.TemplateRole = "test-group"

	role, isContainer := defaultChildrenRole(&outbox)
	require.Equal(t, "outbox", role)
	require.True(t, isContainer)

	role, isContainer = defaultChildrenRole(&collection)
	require.Equal(t, "test-collection", role)
	require.True(t, isContainer)

	// A Template with no model builds Streams
	role, isContainer = defaultChildrenRole(&unnamedModel)
	require.Equal(t, "test-unnamed", role)
	require.True(t, isContainer)

	role, isContainer = defaultChildrenRole(&group)
	require.Empty(t, role)
	require.False(t, isContainer)
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

// TestInvalidDefaultChildren_TemplateOutsideTheOutbox confirms that a Template an outbox cannot hold is refused from an outbox
func TestInvalidDefaultChildren_TemplateOutsideTheOutbox(t *testing.T) {
	require.Equal(t,
		[]string{"Default child names a Template that cannot be placed beneath this one"},
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
			"Default child names a Template that cannot be placed beneath this one",
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
