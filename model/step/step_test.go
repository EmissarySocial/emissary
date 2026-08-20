package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestNew_Dispatch verifies that New() routes every supported "do" value to the correct concrete
// step type. The returned step's Name() is the dispatch fingerprint: a mis-wired case (e.g. "sleep"
// accidentally constructing a Sort) would surface as a Name() mismatch here. The config for each
// case is the minimum required for that constructor to succeed.
//
// Note: a step's Name() is not always equal to its "do" key (e.g. do "set-args" -> Name "set-args",
// do "sleep" -> Name "set-sleep", do "require-password" -> Name "requirePassword"),
// so expectedName is asserted explicitly per case.
func TestNew_Dispatch(t *testing.T) {

	cases := []struct {
		do           string
		config       mapof.Any
		expectedName string
	}{
		{"add", mapof.Any{"form": map[string]any{"type": "layout-vertical"}}, "add"},
		{"add-event", mapof.Any{}, "add-event"},
		{"add-stream", mapof.Any{}, "add-stream"},
		{"as-confirmation", mapof.Any{}, "as-confirmation"},
		{"as-modal", mapof.Any{}, "as-modal"},
		{"as-tooltip", mapof.Any{}, "as-tooltip"},
		{"cache-url", mapof.Any{}, "cache-url"},
		{"delete", mapof.Any{}, "delete"},
		{"delete-archive", mapof.Any{}, "delete-archive"},
		{"delete-attachments", mapof.Any{}, "delete-attachments"},
		{"dump", mapof.Any{}, "dump"},
		{"edit", mapof.Any{}, "edit"},
		{"edit-connection", mapof.Any{}, "edit-connection"},
		{"edit-content", mapof.Any{"format": "HTML"}, "edit-content"},
		{"edit-registration", mapof.Any{}, "edit-registration"},
		{"edit-table", mapof.Any{"form": map[string]any{"type": "layout-vertical"}}, "edit-table"},
		{"edit-template", mapof.Any{}, "edit-template"},
		{"edit-widget", mapof.Any{}, "edit-widget"},
		{"forward-to", mapof.Any{}, "forward-to"},
		{"get-archive", mapof.Any{}, "get-archive"},
		{"halt", mapof.Any{}, "halt"},
		{"if", mapof.Any{}, "if"},
		{"include", mapof.Any{}, "include"},
		{"inline-error", mapof.Any{}, "inline-error"},
		{"inline-save-button", mapof.Any{}, "inline-save-button"},
		{"inline-success", mapof.Any{}, "inline-success"},
		{"make-archive", mapof.Any{}, "make-archive"},
		{"process-content", mapof.Any{}, "process-content"},
		{"process-tags", mapof.Any{}, "process-tags"},
		{"promote-draft", mapof.Any{}, "promote-draft"},
		{"redirect-to", mapof.Any{}, "redirect-to"},
		{"refresh-page", mapof.Any{}, "refresh-page"},
		{"reload-page", mapof.Any{}, "reload-page"},
		{"remove-event", mapof.Any{}, "remove-event"},
		{"require-password", mapof.Any{}, "requirePassword"},
		{"save", mapof.Any{}, "save"},
		{"save-and-publish", mapof.Any{}, "save-and-publish"},
		{"schedule-delete", mapof.Any{}, "schedule-delete"},
		{"search-index", mapof.Any{}, "search-index"},
		{"send-email", mapof.Any{}, "send-email"},
		{"set-args", mapof.Any{}, "set-args"},
		{"set-circle-sharing", mapof.Any{"role": "editor"}, "set-circle-sharing"},
		{"set-data", mapof.Any{}, "set-data"},
		{"set-header", mapof.Any{}, "set-header"},
		{"set-password", mapof.Any{}, "set-password"},
		{"set-privileges", mapof.Any{}, "set-privileges"},
		{"set-query-param", mapof.Any{}, "set-query-param"},
		{"set-response", mapof.Any{}, "set-response"},
		{"set-simple-sharing", mapof.Any{"role": "editor"}, "set-simple-sharing"},
		{"set-state", mapof.Any{"state": "published"}, "set-state"},
		{"set-thumbnail", mapof.Any{}, "set-thumbnail"},
		{"sleep", mapof.Any{}, "set-sleep"},
		{"sort", mapof.Any{}, "set-sort"},
		{"sort-attachments", mapof.Any{}, "sort-attachments"},
		{"sort-widgets", mapof.Any{}, "sort-widgets"},
		{"startup-create-streams", mapof.Any{}, "startup-create-streams"},
		{"startup-save-task", mapof.Any{"value": "sample-content"}, "startup-save-task"},
		{"trigger-event", mapof.Any{}, "trigger-event"},
		{"unpublish", mapof.Any{}, "unpublish"},
		{"upload-attachments", mapof.Any{}, "upload-attachments"},
		{"view-attachment", mapof.Any{"format": []string{"pdf"}}, "view-attachment"},
		{"view-css", mapof.Any{}, "view-css"},
		{"view-feed", mapof.Any{}, "view-feed"},
		{"view-html", mapof.Any{}, "view-html"},
		{"view-json", mapof.Any{"value": ".Object"}, "view-json"},
		{"with-annotation", mapof.Any{}, "with-annotation"},
		{"with-attachment", mapof.Any{}, "with-attachment"},
		{"with-children", mapof.Any{}, "with-children"},
		{"with-circle", mapof.Any{}, "with-circle"},
		{"with-draft", mapof.Any{}, "with-draft"},
		{"with-folder", mapof.Any{}, "with-folder"},
		{"with-follower", mapof.Any{}, "with-follower"},
		{"with-following", mapof.Any{}, "with-following"},
		{"with-import", mapof.Any{}, "with-import"},
		{"with-keypackage", mapof.Any{}, "with-key-package"},
		{"with-merchant-account", mapof.Any{}, "with-merchant-account"},
		{"with-message", mapof.Any{}, "with-message"},
		{"with-next-sibling", mapof.Any{}, "with-next-sibling"},
		{"with-oauth-token", mapof.Any{}, "with-oauth-token"},
		{"with-parent", mapof.Any{}, "with-parent"},
		{"with-prev-sibling", mapof.Any{}, "with-prev-sibling"},
		{"with-privilege", mapof.Any{}, "with-privilege"},
		{"with-response", mapof.Any{}, "with-response"},
		{"with-rule", mapof.Any{}, "with-rule"},
	}

	for _, testCase := range cases {
		t.Run(testCase.do, func(t *testing.T) {

			config := testCase.config
			config["do"] = testCase.do

			step, err := New(config)
			require.Nil(t, err, "New(%q) should not error", testCase.do)
			require.NotNil(t, step)
			require.Equal(t, testCase.expectedName, step.Name(), "do %q dispatched to the wrong step type", testCase.do)
		})
	}
}

// TestNew_UnrecognizedStep verifies that an unknown step name is rejected
func TestNew_UnrecognizedStep(t *testing.T) {
	_, err := New(mapof.Any{"do": "this-step-does-not-exist"})
	require.NotNil(t, err)
}

// TestNew_MissingDo verifies that a step with no "do" key is rejected
func TestNew_MissingDo(t *testing.T) {
	// An empty "do" is unrecognized and returns an error.
	_, err := New(mapof.Any{})
	require.NotNil(t, err)
}
