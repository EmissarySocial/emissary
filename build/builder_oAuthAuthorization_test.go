package build

import (
	"html/template"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReportedBug_OAuthAuthorizePageNo500 is the regression test for the reported
// HTTP 500 on GET /oauth/authorize: the shared includes-head template evaluates
// .IsIndexable, and OAuthAuthorization had no such method, so html/template failed
// with "can't evaluate field IsIndexable".  The builder now embeds Theme, which
// supplies both the method and the FALSE that keeps this page out of the index.
func TestReportedBug_OAuthAuthorizePageNo500(t *testing.T) {

	// The literal conditional from the theme includes-head.html templates.
	head := template.Must(template.New("head").Parse(
		`{{- if not .IsIndexable }}<meta name="robots" content="noindex">{{- end }}`))

	// The reported bug: executing the head against the OAuth builder must not error...
	var buffer strings.Builder
	require.Nil(t, head.Execute(&buffer, OAuthAuthorization{}))

	// ...and the consent page is ephemeral + user-specific, so it must emit "noindex".
	require.Equal(t, `<meta name="robots" content="noindex">`, buffer.String())
	require.False(t, OAuthAuthorization{}.IsIndexable())
}
