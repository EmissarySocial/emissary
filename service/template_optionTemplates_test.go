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

// walkOptionTemplateSteps calls fn for every option template in every edit step, descending
// into container steps and switching objects where with-folder and with-circle do
func walkOptionTemplateSteps(steps []modelStep.Step, object data.Object, fn func(data.Object, form.Element, string, loose.Template)) {

	for _, step := range steps {

		switch typed := step.(type) {

		case modelStep.EditModelObject:
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
