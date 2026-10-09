package model

import (
	"html/template"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// inheritParentHTML defines a template whose escaping rewrites both a pipeline and an attribute
const inheritParentHTML = `{{define "shared"}}<p title="{{.}}">{{.}}</p>{{end}}`

// TestTheme_Inherit_RendersConcurrently confirms two children of one parent Theme can render
// alongside the parent without a data race, which only `go test -race` detects
func TestTheme_Inherit_RendersConcurrently(t *testing.T) {

	parent := NewTheme("parent", template.FuncMap{})
	template.Must(parent.HTMLTemplate.Parse(inheritParentHTML))

	first := NewTheme("first", template.FuncMap{})
	first.Inherit(&parent)

	second := NewTheme("second", template.FuncMap{})
	second.Inherit(&parent)

	renderConcurrently(t, parent.HTMLTemplate, first.HTMLTemplate, second.HTMLTemplate)
}

// TestTemplate_Inherit_RendersConcurrently confirms two children of one parent Template can
// render alongside the parent without a data race, which only `go test -race` detects
func TestTemplate_Inherit_RendersConcurrently(t *testing.T) {

	parent := NewTemplate("parent", template.FuncMap{})
	template.Must(parent.HTMLTemplate.Parse(inheritParentHTML))

	first := NewTemplate("first", template.FuncMap{})
	first.Inherit(&parent)

	second := NewTemplate("second", template.FuncMap{})
	second.Inherit(&parent)

	renderConcurrently(t, parent.HTMLTemplate, first.HTMLTemplate, second.HTMLTemplate)
}

// TestRegistration_Inherit_RendersConcurrently confirms two children of one parent Registration
// can render alongside the parent without a data race, which only `go test -race` detects
func TestRegistration_Inherit_RendersConcurrently(t *testing.T) {

	parent := NewRegistration("parent", template.FuncMap{})
	template.Must(parent.HTMLTemplate.Parse(inheritParentHTML))

	first := NewRegistration("first", template.FuncMap{})
	first.Inherit(&parent)

	second := NewRegistration("second", template.FuncMap{})
	second.Inherit(&parent)

	renderConcurrently(t, parent.HTMLTemplate, first.HTMLTemplate, second.HTMLTemplate)
}

// renderConcurrently executes the "shared" template in every set at once, and requires each render to succeed
func renderConcurrently(t *testing.T, sets ...*template.Template) {

	t.Helper()

	errs := make([]error, len(sets))

	var wg sync.WaitGroup

	for index, set := range sets {
		wg.Go(func() {
			errs[index] = set.ExecuteTemplate(io.Discard, "shared", "<b>&</b>")
		})
	}

	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
}
