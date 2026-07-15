package build

import (
	"html/template"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

func TestOutboxBuilder(t *testing.T) {
	var _ StateSetter = Outbox{}
}

// TestOutbox_IsIndexable proves the profile's "Index on Search Engines" setting
// drives Outbox.IsIndexable, which overrides the indexable-by-default Common.
func TestOutbox_IsIndexable(t *testing.T) {

	// A brand-new/opted-out profile (isIndexable=false) is NOT indexable...
	optedOut := Outbox{_user: &model.User{IsIndexable: false}}
	require.False(t, optedOut.IsIndexable())

	// ...while a profile that opted in IS indexable.
	optedIn := Outbox{_user: &model.User{IsIndexable: true}}
	require.True(t, optedIn.IsIndexable())

	// Every other page type inherits the Common default: indexable.
	require.True(t, Common{}.IsIndexable())
}

// TestReportedBug_ProfileNoIndexDirective ties the fix back to the report: a
// public profile with isIndexable=false must emit a "noindex" robots directive
// in the page <head>, and an indexable profile must not. This exercises the
// exact conditional the theme includes-head.html templates render.
func TestReportedBug_ProfileNoIndexDirective(t *testing.T) {

	head := template.Must(template.New("head").Parse(
		`{{- if not .IsIndexable }}<meta name="robots" content="noindex">{{- end }}`))

	render := func(builder Builder) string {
		var buffer strings.Builder
		require.Nil(t, head.Execute(&buffer, builder))
		return buffer.String()
	}

	// The reported bug: opting out must now produce the directive.
	optedOut := render(Outbox{_user: &model.User{IsPublic: true, IsIndexable: false}})
	require.Equal(t, `<meta name="robots" content="noindex">`, optedOut)

	// Opting in emits nothing (default-indexable, no directive).
	optedIn := render(Outbox{_user: &model.User{IsPublic: true, IsIndexable: true}})
	require.Equal(t, "", optedIn)
}
