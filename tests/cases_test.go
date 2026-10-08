package tests

import (
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// postCase is one step that applies POSTed values to an object, in the context its
// builder gives it.  Apply reproduces the step's Post method with the same library calls.
type postCase struct {
	Name      string                                    // Stable identifier, used as the golden key
	Fields    []postField                               // Every value the step reads from the request
	Schema    schema.Schema                             // Describes each field, and reads results back
	NewObject func() any                                // Returns a fresh object for the step to change
	Apply     func(object any, values url.Values) error // The step's Post, against that object

	// A step that ranges over Go maps sets Orders, which returns the keys of each map, and
	// ApplyOrdered, which runs the step with each map visited in the order given
	Orders       func(values url.Values) [][]string
	ApplyOrdered func(object any, values url.Values, orders [][]string) error
}

// postField is one value a step reads from the request
type postField struct {
	Path       string         // Request key, which is also the schema path unless SchemaPath is set
	SchemaPath string         // Schema path, when it differs from the request key
	Widget     string         // Form widget type, or the step's own name for steps without a form
	Element    schema.Element // The schema element at Path, or nil when the schema does not define it
	Enum       []string       // Values that the step accepts, when they are known without a builder
}

// collection holds every postCase in a template group, and every POST-reading step that
// this suite does not pin, with the reason
type collection struct {
	Cases     []postCase
	SetData   []setDataUse
	Inventory map[string]string
}

// collectGroup walks every template, widget, and registration in a loaded group
func collectGroup(group *loadedGroup) collection {

	result := collection{Inventory: map[string]string{}}

	// Template actions, in each context the template's builder provides
	for _, templateID := range group.templateIDs() {

		template, err := group.Templates.Load(templateID)

		if err != nil {
			result.Inventory["template:"+templateID] = "does not load: " + err.Error()
			continue
		}

		for _, context := range templateContexts(group, template) {
			for _, actionID := range sortedKeys(template.Actions) {
				prefix := "template:" + templateID + "/" + actionID
				if context.Label != "" {
					prefix += "@" + context.Label
				}
				result.walk(group, prefix, template.Actions[actionID].Steps, context)
			}
		}
	}

	// Widget settings forms, which edit-widget applies to the StreamWidget's data
	for _, lookup := range group.Widgets.List() {
		widget, _ := group.Widgets.Get(lookup.Value)
		result.Cases = append(result.Cases, editWidgetCase(widget))
		result.walk(group, "widget:"+widget.WidgetID+"/saveSteps", widget.SaveSteps, widgetContext(widget))
	}

	// Registration settings forms, and the registration's own actions against a new User
	for _, lookup := range group.Registrations.List() {

		registration, err := group.Registrations.Load(lookup.Value)

		if err != nil {
			result.Inventory["registration:"+lookup.Value] = "does not load: " + err.Error()
			continue
		}

		result.Cases = append(result.Cases, editRegistrationCase(registration))

		userContext := builderContext{Schema: schema.New(model.UserSchema()), NewObject: newPointer(model.NewUser)}
		for _, actionID := range sortedKeys(registration.Actions) {
			result.walk(group, "registration:"+registration.RegistrationID+"/"+actionID, registration.Actions[actionID].Steps, userContext)
		}
	}

	sort.Slice(result.Cases, func(i, j int) bool { return result.Cases[i].Name < result.Cases[j].Name })
	sort.Slice(result.SetData, func(i, j int) bool { return result.SetData[i].Name < result.SetData[j].Name })
	return result
}

// walk visits every step in a pipeline, and recurses into every composite step
func (c *collection) walk(group *loadedGroup, prefix string, steps []step.Step, context builderContext) {

	for index, current := range steps {

		name := prefix + "/" + strconv.Itoa(index) + ":" + current.Name()

		// Pin each step that applies POST values through form or schema
		switch typed := current.(type) {

		case step.SetData:
			c.SetData = append(c.SetData, setDataUse{Name: name, Step: typed, Context: context})

		case step.TableEditor:
			for _, row := range editTableRows {
				c.Cases = append(c.Cases, editTableCase(name+"?edit="+row, row, typed, context))
			}

		default:
			if postCase, ok := newPostCase(group, name, current, context); ok {
				c.Cases = append(c.Cases, postCase)
			} else if reason, reads := unpinnedPostReaders[current.Name()]; reads {
				c.Inventory[name] = reason
			}
		}

		// Recurse into every []step.Step field, so that no nested step is missed
		for field, subSteps := range nestedSteps(current) {

			subContext, reason := subContext(context, current)

			if reason != "" {
				c.Inventory[name+"."+field] = "not pinned: " + reason
				continue
			}

			subPrefix := name + "." + field
			if subContext.Label != "" && subContext.Label != context.Label {
				subPrefix += "@" + subContext.Label
			}

			c.walk(group, subPrefix, subSteps, subContext)
		}
	}
}

// nestedSteps returns every exported []step.Step field of a step, by field name
func nestedSteps(current step.Step) map[string][]step.Step {

	result := map[string][]step.Step{}
	value := reflect.Indirect(reflect.ValueOf(current))

	if value.Kind() != reflect.Struct {
		return result
	}

	stepSlice := reflect.TypeOf([]step.Step{})

	for index := range value.NumField() {
		field := value.Type().Field(index)
		if field.IsExported() && (field.Type == stepSlice) {
			if subSteps := value.Field(index).Interface().([]step.Step); len(subSteps) > 0 {
				result[field.Name] = subSteps
			}
		}
	}

	return result
}

// unpinnedPostReaders names every step that reads the request body but is not pinned
// here, and why.  The inventory golden lists each occurrence.
var unpinnedPostReaders = map[string]string{
	"add-stream":         "not pinned: sets with-data values rendered from templates, not POSTed fields",
	"delete-attachments": "not pinned: deletes stored attachments; writes only an empty string",
	"edit-connection":    "not pinned: provider forms are defined in Go, not in templates",
	"edit-content":       "not pinned: content is converted by service.Content, not set through form or schema",
	"require-password":   "not pinned: checks a password against the database",
	"set-circle-sharing": "not pinned: reads circle IDs that must exist in the database",
	"set-password":       "not pinned: hashes and stores a password through the User service",
	"set-privileges":     "not pinned: reads privilege IDs that must exist in the database",
	"set-response":       "not pinned: writes a Response record through service.Response",
	"set-simple-sharing": "not pinned: reads group IDs that must exist in the database",
	"set-thumbnail":      "not pinned: writes the ID or URL of a stored attachment",
	"sort":               "not pinned: POST keys are database record IDs",
	"sort-attachments":   "not pinned: POST keys are database record IDs",
	"sort-widgets":       "not pinned: POST keys are widget IDs on a stored Stream",
	"upload-attachments": "not pinned: writes the ID, URL, or filename of an uploaded file",
}

// newPostCase returns the postCase for a step that applies POST values, or FALSE
func newPostCase(group *loadedGroup, name string, current step.Step, context builderContext) (postCase, bool) {

	switch typed := current.(type) {

	case step.EditModelObject:
		return editModelObjectCase(name, typed, context), true

	case step.AddModelObject:
		return addModelObjectCase(name, typed, context), true

	case step.ReadForm:
		return readFormCase(name, typed), true

	case step.EditTemplate:
		return editTemplateCase(group, name, typed, context), true
	}

	return postCase{}, false
}

// editModelObjectCase reproduces build.StepEditModelObject.Post
func editModelObjectCase(name string, current step.EditModelObject, context builderContext) postCase {

	// build.StepEditModelObject.getForm: an empty form falls back to the Domain builder's form
	element := current.Form
	if element.IsEmpty() && !context.PropertyForm.IsEmpty() {
		element = context.PropertyForm
	}

	return postCase{
		Name:      name,
		Fields:    formFields(context.Schema, element),
		Schema:    context.Schema,
		NewObject: context.NewObject,
		Apply: func(object any, values url.Values) error {
			stepForm := form.New(context.Schema, element)
			return stepForm.SetURLValues(object, values, nil)
		},
	}
}

// addModelObjectCase reproduces build.StepAddModelObject.Post, which sets every request key
// as a []string, in map order, and without its "defaults" steps (see AGENTS.md)
func addModelObjectCase(name string, current step.AddModelObject, context builderContext) postCase {

	apply := func(object any, values url.Values, order []string) error {
		for _, key := range order {
			if err := context.Schema.Set(object, key, values[key]); err != nil {
				return err
			}
		}
		return nil
	}

	return postCase{
		Name:      name,
		Fields:    formFields(context.Schema, current.Form),
		Schema:    context.Schema,
		NewObject: context.NewObject,
		Apply: func(object any, values url.Values) error {
			return apply(object, values, sortedKeys(values))
		},
		Orders: func(values url.Values) [][]string {
			return [][]string{sortedKeys(values)}
		},
		ApplyOrdered: func(object any, values url.Values, orders [][]string) error {
			return apply(object, values, orders[0])
		},
	}
}

// editTableRows are the "?edit=" values posted to every edit-table step: rows 0 to 16,
// then the two kinds of index the step rejects
var editTableRows = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "-1", "first"}

// editTableCase reproduces build.StepTableEditor.Post for one "?edit=" row
func editTableCase(name string, row string, current step.TableEditor, context builderContext) postCase {

	fields := make([]postField, 0)

	for _, element := range current.Form.AllElements() {
		field := newPostField(context.Schema, current.Path+"."+row+"."+element.Path, element.Type)
		field.SchemaPath = field.Path
		field.Path = element.Path // The request key is the column path, not the schema path
		fields = append(fields, field)
	}

	return postCase{
		Name:      name,
		Fields:    fields,
		Schema:    context.Schema,
		NewObject: context.NewObject,
		Apply: func(object any, values url.Values) error {

			const location = "build.StepTableEditor.Post"

			// Bounds checking, as the step does before any value is set
			editIndex, ok := convert.IntOk(row, 0)

			if !ok {
				return derp.Internal(location, "Failed to convert edit index", current.Path, row)
			}

			if editIndex < 0 {
				return derp.Internal(location, "Edit index out of range", current.Path, editIndex)
			}

			for _, element := range current.Form.AllElements() {
				if err := context.Schema.Set(object, current.Path+"."+row+"."+element.Path, values.Get(element.Path)); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// readFormCase reproduces build.StepReadForm.Post, which writes into a new mapof.Any rather
// than the builder's object, visiting the schema's properties in map order
func readFormCase(name string, current step.ReadForm) postCase {

	properties := sortedKeys(current.Schema.AllProperties())

	fields := make([]postField, 0)
	for _, path := range properties {
		fields = append(fields, newPostField(current.Schema, path, "read-form"))
	}

	apply := func(object any, values url.Values, order []string) error {

		target := object.(*mapof.Any)

		for _, path := range order {

			value := strings.Join(values[path], ",")

			if element, ok := current.Schema.GetStringElement(path); ok {
				if (element.MaxLength > 0) && (utf8.RuneCountInString(value) > element.MaxLength) {
					return derp.BadRequest("build.StepReadForm.Post", "Value is too long", path, "maximum: "+strconv.Itoa(element.MaxLength))
				}
			}

			if err := current.Schema.Set(target, path, value); err != nil {
				return err
			}
		}

		_, err := current.Schema.Validate(*target)
		return err
	}

	return postCase{
		Name:      name,
		Fields:    fields,
		Schema:    current.Schema,
		NewObject: newPointer(mapof.NewAny),
		Apply: func(object any, values url.Values) error {
			return apply(object, values, properties)
		},
		Orders: func(url.Values) [][]string {
			return [][]string{properties}
		},
		ApplyOrdered: func(object any, values url.Values, orders [][]string) error {
			return apply(object, values, orders[0])
		},
	}
}

// editTemplateCase reproduces build.StepEditTemplate.Post, including its allow-list
func editTemplateCase(group *loadedGroup, name string, current step.EditTemplate, context builderContext) postCase {

	fields := make([]postField, 0)

	for _, path := range current.Paths {
		field := newPostField(context.Schema, path, "edit-template")
		field.Enum = allowedTemplates(group, path, nil)
		fields = append(fields, field)
	}

	return postCase{
		Name:      name,
		Fields:    fields,
		Schema:    context.Schema,
		NewObject: context.NewObject,
		Apply: func(object any, values url.Values) error {
			for _, path := range current.Paths {
				newTemplateID := values.Get(path)
				if containsString(allowedTemplates(group, path, object), newTemplateID) {
					if err := context.Schema.Set(object, path, newTemplateID); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

// allowedTemplates reproduces build.StepEditTemplate.listTemplates
func allowedTemplates(group *loadedGroup, path string, object any) []string {

	result := make([]string, 0)

	switch path {

	case "templateId":
		if stream, ok := object.(*model.Stream); ok {
			if parentTemplate, err := group.Templates.Load(stream.ParentTemplateID); err == nil {
				for _, lookup := range group.Templates.ListByContainer(parentTemplate.TemplateRole) {
					result = append(result, lookup.Value)
				}
			}
		}

	case "inboxTemplate":
		for _, lookup := range group.Templates.ListByTemplateRole("user-inbox") {
			result = append(result, lookup.Value)
		}

	case "outboxTemplate":
		for _, lookup := range group.Templates.ListByTemplateRole("user-outbox") {
			result = append(result, lookup.Value)
		}
	}

	sort.Strings(result)
	return result
}

// editWidgetCase reproduces build.StepEditWidget.Post for one widget definition
func editWidgetCase(widget model.Widget) postCase {

	return postCase{
		Name:      "widget:" + widget.WidgetID + "/edit-widget",
		Fields:    formFields(widget.Schema, widget.Form),
		Schema:    widget.Schema,
		NewObject: newPointer(mapof.NewAny),
		Apply: func(object any, values url.Values) error {
			widgetForm := form.New(widget.Schema, widget.Form)
			return widgetForm.SetURLValues(object, values, nil)
		},
	}
}

// editRegistrationCase reproduces build.StepEditRegistration.Post for one registration,
// which applies the posted values through the registration's schema, not its form
func editRegistrationCase(registration model.Registration) postCase {

	return postCase{
		Name:      "registration:" + registration.RegistrationID + "/edit-registration",
		Fields:    formFields(registration.Schema, registration.Form),
		Schema:    registration.Schema,
		NewObject: newPointer(mapof.NewString),
		Apply: func(object any, values url.Values) error {
			return registration.Schema.SetURLValues(object, registrationInputs(values))
		},
		Orders: func(values url.Values) [][]string {
			return [][]string{sortedKeys(registrationInputs(values))}
		},
		ApplyOrdered: func(object any, values url.Values, orders [][]string) error {
			return setURLValuesInOrder(registration.Schema, object, registrationInputs(values), orders[0])
		},
	}
}

// registrationInputs copies the request, without the "registrationId" control parameter
func registrationInputs(values url.Values) url.Values {
	result := url.Values{}
	for key, value := range values {
		result[key] = value
	}
	result.Del("registrationId")
	return result
}

// setURLValuesInOrder reproduces rosetta's Schema.SetURLValues, visiting the keys in the
// given order instead of Go's map order
func setURLValuesInOrder(s schema.Schema, object any, values url.Values, order []string) error {

	const location = "schema.Schema.SetURLValues"

	for _, path := range order {
		if err := s.Set(object, path, values[path]); err != nil {
			return derp.Wrap(err, location, "Setting value", path)
		}
	}

	if err := s.ValidateRequiredIf(object); err != nil {
		return derp.Wrap(err, location, "Validating values")
	}

	return nil
}

// formFields lists every input in a form, including the sub-fields a place widget posts
func formFields(s schema.Schema, element form.Element) []postField {

	result := make([]postField, 0)

	for _, input := range element.AllElements() {

		field := newPostField(s, input.Path, input.Type)

		if enum, ok := input.Options["enum"].([]form.LookupCode); ok {
			for _, code := range enum {
				field.Enum = append(field.Enum, code.Value)
			}
		}

		result = append(result, field)

		// widget.Place.SetURLValue reads three sub-fields instead of its own path
		if input.Type == "place" {
			for _, suffix := range []string{"formatted", "longitude", "latitude"} {
				result = append(result, newPostField(s, input.Path+"."+suffix, "place."+suffix))
			}
		}
	}

	return result
}

// newPostField describes one request key against a schema
func newPostField(s schema.Schema, path string, widget string) postField {

	result := postField{Path: path, Widget: widget}

	if element, ok := s.GetElement(path); ok {
		result.Element = element
	}

	return result
}

// sortedKeys returns the keys of any string-keyed map, sorted
func sortedKeys[V any, M ~map[string]V](value M) []string {
	result := make([]string, 0, len(value))
	for key := range value {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

// containsString reports whether a slice holds a value
func containsString(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
