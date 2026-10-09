package tests

import (
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"text/template/parse"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/rosetta/compare"
	"github.com/stretchr/testify/require"
)

// setDataUse is one set-data step, in the context its builder gives it
type setDataUse struct {
	Name    string
	Step    step.SetData
	Context builderContext
}

// setDataGolden is the pinned behavior of every set-data step in one template group
type setDataGolden struct {
	Uses map[string]setDataPinned `json:"uses"`
}

// setDataPinned is the pinned behavior of one set-data step
type setDataPinned struct {
	Fields   []string          `json:"fields"`             // Every path the step can write, and where its value comes from
	Values   map[string]string `json:"values,omitempty"`   // The template source of each "values" entry
	Defaults map[string]string `json:"defaults,omitempty"` // Each "defaults" entry, as "type value"
	Results  map[string]result `json:"results"`            // "METHOD/scenario" -> what it did
}

// setDataScenario is one combination of request input and rendered "values"
type setDataScenario struct {
	Name     string
	Request  func(fields []postField) url.Values // The query and body posted
	Rendered func(field postField, source *template.Template) string
	Prefill  bool // TRUE to set every path the step writes before it runs
}

// setDataScenarios reuse the form-POST scenarios, and supply the rendered text of each
// "values" template (see AGENTS.md, "set-data")
var setDataScenarios = []setDataScenario{
	{Name: "literal", Request: emptyRequest, Rendered: renderLiteral},
	{Name: "typical", Request: eachField(typicalValue), Rendered: func(field postField, _ *template.Template) string { return typicalValue(field)[0] }},
	{Name: "empty", Request: eachField(func(postField) []string { return []string{""} }), Rendered: func(postField, *template.Template) string { return "" }},
	{Name: "hostile", Request: eachField(hostileValue), Rendered: func(field postField, _ *template.Template) string { return hostileValue(field)[0] }},
	{Name: "prefilled", Request: emptyRequest, Rendered: renderLiteral, Prefill: true},
}

// TestSetData pins every set-data step in every template, widget, and registration, under
// GET and POST, against the set-data golden file for its template group
func TestSetData(t *testing.T) {

	for _, group := range templateGroups {
		t.Run(group.Name, func(t *testing.T) {

			loaded := loadGroup(t, group)
			actual := pinSetData(loaded)
			filename := filepath.Join("testdata", group.Name+".setdata.golden.json")

			if *update {
				require.Equal(t, actual, pinSetData(loaded), "behavior is not deterministic, so it cannot be pinned")

				encoded, err := json.MarshalIndent(actual, "", "\t")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filename, append(encoded, '\n'), 0o600))
				return
			}

			encoded, err := os.ReadFile(filename) // #nosec G304 -- the name is built from a fixed group list
			require.NoError(t, err, "missing golden file; run go test ./tests -update")

			expected := setDataGolden{}
			require.NoError(t, json.Unmarshal(encoded, &expected))
			require.Equal(t, sortedKeys(expected.Uses), sortedKeys(actual.Uses), "the set of set-data steps changed")

			for _, name := range sortedKeys(expected.Uses) {
				t.Run(name, func(t *testing.T) {
					require.Equal(t, expected.Uses[name], actual.Uses[name])
				})
			}
		})
	}
}

// pinSetData runs every scenario, under both methods, against every set-data step in a group
func pinSetData(group *loadedGroup) setDataGolden {

	pinnedGroup := setDataGolden{Uses: map[string]setDataPinned{}}

	for _, use := range collectGroup(group).SetData {

		pinned := setDataPinned{
			Fields:  use.sortedFields(),
			Results: map[string]result{},
		}

		for _, key := range sortedKeys(use.Step.Values) {
			if pinned.Values == nil {
				pinned.Values = map[string]string{}
			}
			kind := " (renders against the builder; mocked)"
			if isLiteral(use.Step.Values[key]) {
				kind = " (literal)"
			}
			pinned.Values[key] = use.Step.Values[key].Root.String() + kind
		}

		for _, key := range sortedKeys(use.Step.Defaults) {
			if pinned.Defaults == nil {
				pinned.Defaults = map[string]string{}
			}
			pinned.Defaults[key] = describe(use.Step.Defaults[key])
		}

		for _, method := range []string{"GET", "POST"} {
			for _, scenario := range setDataScenarios {
				postCase := use.postCase(method, scenario)
				pinned.Results[method+"/"+scenario.Name] = pinScenario(postCase, scenario.Request(postCase.Fields))
			}
		}

		pinnedGroup.Uses[use.Name] = pinned
	}

	return pinnedGroup
}

// postCase adapts one method and scenario of a set-data step to the shared runner
func (use setDataUse) postCase(method string, scenario setDataScenario) postCase {

	fields := use.fields()
	request := make([]postField, 0)
	for _, field := range fields {
		if (field.Widget == "from-url") || ((field.Widget == "from-form") && (method == "POST")) {
			request = append(request, field)
		}
	}

	return postCase{
		Name:      use.Name,
		Fields:    request,
		Schema:    use.Context.Schema,
		NewObject: use.newObject(scenario.Prefill, fields),
		Apply: func(object any, values url.Values) error {
			return use.apply(method, scenario, object, values, sortedKeys(use.Step.Values), sortedKeys(use.Step.Defaults))
		},
		Orders: func(url.Values) [][]string {
			return [][]string{sortedKeys(use.Step.Values), sortedKeys(use.Step.Defaults)}
		},
		ApplyOrdered: func(object any, values url.Values, orders [][]string) error {
			return use.apply(method, scenario, object, values, orders[0], orders[1])
		},
	}
}

// apply reproduces build.StepSetData.Get (for GET) and build.StepSetData.Post (for POST),
// visiting "values" and "defaults" in the orders given
func (use setDataUse) apply(method string, scenario setDataScenario, object any, values url.Values, valuesOrder []string, defaultsOrder []string) error {

	s := use.Context.Schema

	// setURLPaths: only non-empty query values are set
	for _, path := range use.Step.FromURL {
		if value := values.Get(path); value != "" {
			if err := s.Set(object, path, value); err != nil {
				return err
			}
		}
	}

	// FromForm is read on POST only, and every value is joined with commas
	if method == "POST" {
		for _, path := range use.Step.FromForm {
			if err := s.Set(object, path, strings.Join(values[path], ",")); err != nil {
				return err
			}
		}
	}

	// Values range over a map in production; the caller chooses the order here
	for _, key := range valuesOrder {
		rendered := scenario.Rendered(newPostField(s, key, "values"), use.Step.Values[key])
		if err := s.Set(object, key, rendered); err != nil {
			return err
		}
	}

	// Defaults read the current value through the BUILDER, which implements none of the
	// schema getter interfaces, so the read always fails and every default is applied
	for _, name := range defaultsOrder {
		value := use.Step.Defaults[name]
		if currentValue, _ := s.Get(builderStandIn{}, name); compare.IsZero(currentValue) {
			if err := s.Set(object, name, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// builderStandIn stands in for a Builder in schema.Get.  No production builder implements
// a schema getter interface; TestEmulatedSources fails if one ever does.
type builderStandIn struct{}

// fields lists every path a set-data step writes, and where its value comes from
func (use setDataUse) fields() []postField {

	result := make([]postField, 0)

	for _, path := range use.Step.FromURL {
		result = append(result, newPostField(use.Context.Schema, path, "from-url"))
	}

	for _, path := range use.Step.FromForm {
		result = append(result, newPostField(use.Context.Schema, path, "from-form"))
	}

	for _, key := range sortedKeys(use.Step.Values) {
		result = append(result, newPostField(use.Context.Schema, key, "values"))
	}

	for _, key := range sortedKeys(use.Step.Defaults) {
		result = append(result, newPostField(use.Context.Schema, key, "defaults"))
	}

	return result
}

// sortedFields describes every path the step writes, which pins the step's definition
func (use setDataUse) sortedFields() []string {
	return postCase{Fields: use.fields()}.sortedFields()
}

// newObject returns a constructor for the step's object, optionally with every path the
// step writes already holding a typical value, so that "defaults" have something to keep
func (use setDataUse) newObject(prefill bool, fields []postField) func() any {

	return func() any {

		object := use.Context.NewObject()

		if prefill {
			for _, field := range fields {
				_ = use.Context.Schema.Set(object, field.Path, typicalValue(field)[0]) // A path the schema rejects stays empty, which the golden shows
			}
		}

		return object
	}
}

// emptyRequest posts nothing at all
func emptyRequest([]postField) url.Values {
	return url.Values{}
}

// renderLiteral renders a "values" template with no data, the way build.executeTemplate
// does: a template that fails to render produces an empty string
func renderLiteral(_ postField, source *template.Template) string {

	var buffer bytes.Buffer

	if err := source.Execute(&buffer, nil); err != nil {
		return ""
	}

	return buffer.String()
}

// isLiteral reports whether a template has no actions, so that renderLiteral is exact
func isLiteral(source *template.Template) bool {

	for _, node := range source.Root.Nodes {
		if node.Type() != parse.NodeText {
			return false
		}
	}

	return true
}

// describe formats a defaults value the way the golden files format stored values
func describe(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
