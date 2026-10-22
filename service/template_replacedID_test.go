package service

import (
	"fmt"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/set"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/derp"
	"github.com/benpate/derp/plugins"
	"github.com/stretchr/testify/require"
)

// capturingReporter records every error that derp.Report sends to it
type capturingReporter struct {
	lock   sync.Mutex
	errors []error
}

// Report implements derp.Reporter
func (reporter *capturingReporter) Report(err error) {
	reporter.lock.Lock()
	defer reporter.lock.Unlock()
	reporter.errors = append(reporter.errors, err)
}

// definitionWithLabel returns a minimal Template definition for the shared ID, labeled to tell copies apart
func definitionWithLabel(label string) fstest.MapFS {
	return fstest.MapFS{
		"template.hjson": &fstest.MapFile{Data: []byte(`{
			templateId: shared-id
			category: Special
			model: Stream
			label: "` + label + `"
			states: {
				default: {
					label: "Default"
				}
			}
			actions: {}
		}`)},
	}
}

// TestTemplate_ReplacedIDReported confirms that a second definition of one ID is reported and still wins
func TestTemplate_ReplacedIDReported(t *testing.T) {

	reporter := &capturingReporter{}
	derp.SetPlugins(reporter)
	t.Cleanup(func() { derp.SetPlugins(plugins.JSON{}) })

	templateService := &Template{
		templatePrep: make(set.Map[model.Template]),
		funcMap:      emissarytemplates.FuncMap(nullIconProvider{}),
	}

	// The first definition reports nothing
	first := definitionWithLabel("First")
	_, definition := findDefinition(first)
	require.NoError(t, templateService.Add("folder-one", first, definition))
	require.Empty(t, reporter.errors)

	// The second definition is reported with both folders, and replaces the first
	second := definitionWithLabel("Second")
	_, definition = findDefinition(second)
	require.NoError(t, templateService.Add("folder-two", second, definition))

	require.Len(t, reporter.errors, 1)
	details := fmt.Sprint(derp.Details(reporter.errors[0]))
	require.Contains(t, details, "templateId: shared-id")
	require.Contains(t, details, "replaced: folder-one")
	require.Contains(t, details, "replacedBy: folder-two")

	require.Equal(t, "Second", templateService.templatePrep["shared-id"].Label)
}
