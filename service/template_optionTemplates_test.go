package service

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	modelStep "github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/data"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/loose"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedTemplates_OptionTemplates renders every form option template in every embedded
// Template against the object its edit step passes to the form, and requires the object's ID
func TestEmbeddedTemplates_OptionTemplates(t *testing.T) {

	// BUG-204: option templates render against the edited object, not the Builder, so a
	// Builder-only token (like .ObjectID) or a raw ObjectID field (which prints as
	// ObjectID("...")) produces a broken URL.
	files, err := filepath.Glob("../_embed/templates/*/template.hjson")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	rendered := 0

	for _, filename := range files {

		definition, err := os.ReadFile(filename)
		require.NoError(t, err)

		template := model.NewTemplate(filepath.Base(filepath.Dir(filename)), nil)
		require.NoError(t, hjson.Unmarshal(definition, &template), filename)

		for actionID, action := range template.Actions {
			walkOptionTemplateSteps(action.Steps, optionTemplateObject(template.Model), func(object data.Object, element form.Element, key string, optionTemplate loose.Template) {

				where := filename + " " + actionID + " " + element.Path + "." + key
				require.NotNil(t, object, "%s: no edited object for model %q", where, template.Model)
				require.True(t, optionTemplate.IsTemplate(), "%s: option template does not compile: %s", where, optionTemplate)

				result, err := optionTemplate.Execute(object)
				require.NoError(t, err, where)
				require.Contains(t, result, object.ID(), where)
				require.NotContains(t, result, "ObjectID(", where)
				rendered++
			})
		}
	}

	// The ten shipped option templates, so that a walker that finds nothing cannot pass
	require.Equal(t, 10, rendered)
}

// TestEmbeddedForms_NoOtherOptionTemplates requires that forms rendered against something other
// than a model object hold no option templates, until a test covers what they render against
func TestEmbeddedForms_NoOtherOptionTemplates(t *testing.T) {

	// BUG-204: a table form renders against each row, a Theme form against the Domain, a Widget
	// form against its data map, and a Registration form against the new User's data. `{{.ID}}`
	// means something different, or nothing, in each, so an option template there needs a test of its own.
	found := make([]string, 0)

	record := func(where string) func(data.Object, form.Element, string, loose.Template) {
		return func(_ data.Object, element form.Element, key string, _ loose.Template) {
			found = append(found, where+" "+element.Path+"."+key)
		}
	}

	// Table editors in every Template action
	templates, err := filepath.Glob("../_embed/templates/*/template.hjson")
	require.NoError(t, err)
	require.NotEmpty(t, templates)

	for _, filename := range templates {

		definition, err := os.ReadFile(filename)
		require.NoError(t, err)

		template := model.NewTemplate(filepath.Base(filepath.Dir(filename)), nil)
		require.NoError(t, hjson.Unmarshal(definition, &template), filename)

		for actionID, action := range template.Actions {
			walkOtherStepForms(action.Steps, func(element form.Element) {
				walkOptionTemplateElement(element, nil, record(filename+" "+actionID))
			})
		}
	}

	// The settings forms of every Theme, Widget, and Registration
	definitions := 0

	for _, pattern := range []string{"theme.hjson", "widget.hjson", "registration.hjson"} {

		files, err := filepath.Glob("../_embed/templates/*/" + pattern)
		require.NoError(t, err)

		for _, filename := range files {

			contents, err := os.ReadFile(filename)
			require.NoError(t, err)

			definition := struct {
				Form form.Element `json:"form"`
			}{}

			require.NoError(t, hjson.Unmarshal(contents, &definition), filename)
			walkOptionTemplateElement(definition.Form, nil, record(filename))
			definitions++
		}
	}

	// Every definition file was read, so that a walker that finds nothing cannot pass
	require.Positive(t, definitions)
	require.Empty(t, found, "option templates outside edit-model-object and add-model-object need their own render test")
}

// walkOtherStepForms calls fn with the form of every step that renders one against something other
// than the model object, descending into container steps
func walkOtherStepForms(steps []modelStep.Step, fn func(form.Element)) {

	for _, step := range steps {

		switch step.(type) {

		// These render against the model object, and TestEmbeddedTemplates_OptionTemplates covers them
		case modelStep.EditModelObject, modelStep.AddModelObject:

		default:
			if getter, ok := step.(modelStep.FormGetter); ok {
				fn(getter.GetForm())
			}
		}

		// Descend into any container
		if subSteps := reflect.ValueOf(step).FieldByName("SubSteps"); subSteps.IsValid() {
			if children, ok := subSteps.Interface().([]modelStep.Step); ok {
				walkOtherStepForms(children, fn)
			}
		}
	}
}

// walkOptionTemplateSteps calls fn for every option template in every edit step, descending
// into container steps and switching objects where with-folder and with-circle do
func walkOptionTemplateSteps(steps []modelStep.Step, object data.Object, fn func(data.Object, form.Element, string, loose.Template)) {

	for _, step := range steps {

		switch typed := step.(type) {

		case modelStep.EditModelObject:
			walkOptionTemplateElement(typed.Form, object, fn)

		// A new object renders its form exactly as an edited one does
		case modelStep.AddModelObject:
			walkOptionTemplateElement(typed.Form, object, fn)

		case modelStep.WithFolder:
			folder := model.NewFolder()
			walkOptionTemplateSteps(typed.SubSteps, &folder, fn)
			continue

		case modelStep.WithCircle:
			circle := model.NewCircle()
			walkOptionTemplateSteps(typed.SubSteps, &circle, fn)
			continue
		}

		// Descend into any other container, which keeps the same object
		if subSteps := reflect.ValueOf(step).FieldByName("SubSteps"); subSteps.IsValid() {
			if children, ok := subSteps.Interface().([]modelStep.Step); ok {
				walkOptionTemplateSteps(children, object, fn)
			}
		}
	}
}

// walkOptionTemplateElement calls fn for every option template in a form, at any depth, including
// a malformed one, which parsing keeps as a plain string
func walkOptionTemplateElement(element form.Element, object data.Object, fn func(data.Object, form.Element, string, loose.Template)) {

	for key, value := range element.Options {
		switch typed := value.(type) {

		case loose.Template:
			fn(object, element, key, typed)

		case string:
			if strings.Contains(typed, "{{") && strings.Contains(typed, "}}") {
				fn(object, element, key, loose.NewTemplate(typed))
			}
		}
	}

	for _, child := range element.Children {
		walkOptionTemplateElement(child, object, fn)
	}
}

// optionTemplateObject returns the object a Template's builder edits, for the models whose
// Templates use option templates
func optionTemplateObject(modelName string) data.Object {

	switch strings.ToLower(modelName) {

	case "stream":
		stream := model.NewStream()
		return &stream

	case "group":
		group := model.NewGroup()
		return &group

	case "tag":
		searchTag := model.NewSearchTag()
		return &searchTag
	}

	return nil
}
