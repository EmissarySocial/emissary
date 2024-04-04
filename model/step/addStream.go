package step

import (
	"text/template"

	"github.com/EmissarySocial/emissary/tools/val"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// AddStream is an action that can add new sub-streams to the domain.
//
// Uses:
// Display a pop-up to choose a template and create a new stream
// Embed a custom "create" widget into the current page - possibly selecting between multiple templates
// Create a new stream using a specific template as a part of a larger pipeline
type AddStream struct {
	Style         string                        // Style of input widget to use. Options are: "chooser", "modal", and "inline".  Defaults to "chooser".
	Title         string                        // Title to use on the create modal. Defaults to "Add a Stream"
	Location      string                        // Options are: "top", "child", "outbox".  Defaults to "child".
	TemplateID    string                        // ID of the template to use.  If empty, then template roles are used.
	TemplateRoles []string                      // List of acceptable Template Roles that can be used to make a stream.  If empty, then all template for this container are valid.
	WithData      map[string]*template.Template // Map of values to preset in the new stream
}

// NewAddStream returns a fully initialized AddStream record
func NewAddStream(stepInfo mapof.Any) (AddStream, error) {

	// Parse the "with-data" map
	templates := stepInfo.GetMap("with-data")
	withDataMap := make(map[string]*template.Template, len(templates))

	for key := range templates {
		valueTemplate := templates.GetString(key)
		value, err := template.New("value").Parse(valueTemplate)

		if err != nil {
			return AddStream{}, derp.Wrap(err, "model.step.NewAddStream", "Error parsing template", key, valueTemplate)
		}

		withDataMap[key] = value
	}

	// Create the step
	result := AddStream{
		Style:         first(stepInfo.GetString("style"), "chooser"),
		Title:         first(stepInfo.GetString("title"), "Add a Stream"),
		Location:      val.Enum(stepInfo.GetString("location"), "top", "child", "outbox"),
		TemplateID:    stepInfo.GetString("template"),
		TemplateRoles: stepInfo.GetSliceOfString("roles"),
		WithData:      withDataMap,
	}

	return result, nil
}

// AmStep is here to verify that this struct is a build pipeline step
func (step AddStream) AmStep() {}
