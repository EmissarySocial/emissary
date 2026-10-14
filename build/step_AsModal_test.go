package build

import (
	"testing"
	"text/template"

	"github.com/EmissarySocial/emissary/model/step"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// asModalBuilder is a Builder that supplies only IsNew, which the option templates below read.
// Every other Builder method is unused, and panics if called.
type asModalBuilder struct {
	Builder
	isNew bool
}

// IsNew returns the configured value
func (builder asModalBuilder) IsNew() bool {
	return builder.isNew
}

// TestStepAsModal_Options verifies that "as-modal" executes each option against the builder,
// with the server's helper functions, before it wraps the modal (BUG-268)
func TestStepAsModal_Options(t *testing.T) {

	parsed, err := step.NewAsModal(mapof.Any{
		"options": []string{"class:{{iif .IsNew `` `large`}}", "id:notes"},
	}, template.FuncMap(emissarytemplates.FuncMap(nil)))
	require.NoError(t, err)

	modal := StepAsModal(parsed)

	require.Equal(t, []string{"class:", "id:notes"}, modal.options(asModalBuilder{isNew: true}))
	require.Equal(t, []string{"class:large", "id:notes"}, modal.options(asModalBuilder{isNew: false}))
}
