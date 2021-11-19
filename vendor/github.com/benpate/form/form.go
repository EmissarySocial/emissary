package form

import (
	"encoding/json"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/schema"
)

// Form defines a single form element, or a nested form layout.  It can be serialized to and from a database.
type Form struct {
	Path        string            `json:"path"`        // Path to the data value displayed in for this form element
	Kind        string            `json:"kind"`        // The kind of form element
	ID          string            `json:"id"`          // DOM ID to use for this element.
	Label       string            `json:"label"`       // Short label to be displayed on the form element
	Description string            `json:"description"` // Longer description text to be displayed on the form element
	CSSClass    string            `json:"cssClass"`    // CSS Class override to apply to this widget.  This should be used sparingly
	Options     map[string]string `json:"options"`     // Additional custom properties defined by individual widgets
	Rules       map[string]string `json:"rules"`       // Visibility rules (in hyperscript) to apply to UI.
	Children    []Form            `json:"children"`    // Array of sub-form elements that may be displayed depending on the kind.
}

// Parse attempts to convert any value into a Form.
func Parse(data interface{}) (Form, error) {

	var result Form

	switch data := data.(type) {

	case map[string]interface{}:
		err := result.UnmarshalMap(data)
		return result, err

	case []byte:
		err := json.Unmarshal(data, &result)
		return result, err

	case string:
		err := json.Unmarshal([]byte(data), &result)
		return result, err

	}

	return result, derp.New(derp.CodeInternalError, "form.Parse", "Cannot Parse Value: Unknown Datatype", data)
}

// MustParse guarantees that a value has been parsed into a Form, or else it panics the application.
func MustParse(data interface{}) Form {

	result, err := Parse(data)

	if err != nil {
		panic(err)
	}

	return result
}

// UnmarshalMap parses data from a generic structure (map[string]interface{}) into a Form record.
func (form *Form) UnmarshalMap(data map[string]interface{}) error {

	form.Path = convert.String(data["path"])
	form.Kind = convert.String(data["kind"])
	form.ID = convert.String(data["id"])
	form.Label = convert.String(data["label"])
	form.Description = convert.String(data["description"])
	form.CSSClass = convert.String(data["cssClass"])

	form.Options = make(map[string]string)
	if options, ok := data["options"].(map[string]interface{}); ok {
		for key, value := range options {
			form.Options[key] = convert.String(value)
		}
	}

	form.Rules = make(map[string]string)
	if rules, ok := data["rules"].(map[string]interface{}); ok {
		for key, value := range rules {
			form.Rules[key] = convert.String(value)
		}
	}

	if children, ok := data["children"].([]interface{}); ok {
		form.Children = make([]Form, len(children))
		for index, childInterface := range children {
			if childData, ok := childInterface.(map[string]interface{}); ok {
				var childForm Form
				childForm.UnmarshalMap(childData)
				form.Children[index] = childForm
			} else {
				return derp.New(derp.CodeInternalError, "form.UnmarshalMap", "Error parsing child form information.", childInterface)
			}
		}
	}

	return nil
}

// 	Autocomplete string `json:"autocomplete"` // https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/autocomplete

// HTML returns a populated HTML string for the provided form, schema, and value
func (form Form) HTML(library Library, schema *schema.Schema, value interface{}) (string, error) {

	b := html.New()

	if err := form.Write(library, schema, value, b); err != nil {
		return "", derp.Wrap(err, "form.HTML", "Error rendering element", form)
	}

	return b.String(), nil
}

// Write generates an HTML string for the fully populated form into the provided string builder
func (form Form) Write(library Library, schema *schema.Schema, value interface{}, b *html.Builder) error {

	// Try to locate the Renderer in the library
	renderer, err := library.Renderer(form.Kind)

	if err != nil {
		return derp.Wrap(err, "form.Write", "Renderer Not Defined", form)
	}

	// try to render the form into the
	if err := renderer(form, schema, value, b); err != nil {
		return derp.Wrap(err, "form.Write", "Error rendering element", form)
	}

	return nil
}

// AllPaths returns pointers to all of the valid paths in this form
func (form Form) AllPaths() []*Form {

	var result []*Form

	// If THIS element has a Path, then add it to the result
	if form.Path != "" {
		result = []*Form{&form}
	} else {
		result = []*Form{}
	}

	// Scan all chiild elements for THEIR paths, and add them to the result
	for _, child := range form.Children {
		result = append(result, child.AllPaths()...)
	}

	// Success
	return result
}
