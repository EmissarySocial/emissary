package model

// Form defines the structure of a Form to be displayed to the user.  Implementation
// (such as HTML form elements) is left to the client
type Form struct {
	Title       string    `json:"title"              bson:"title"`       // The title of the Form, displayed in large text at the top of the form UX
	Description string    `json:"description"        bson:"description"` // Description of the Form, displayed in normal-size text below the title
	Sections    []Section `json:"sections,omitempty" bson:"sections"`    // Sections of the Form.  These contain actual form fields
}

// Section represents a part of a Form that has its own label
type Section struct {
	Label       string  `json:"label"                 bson:"label"`                 // Section label.  Displayed at the top of the section in large text
	Description string  `json:"description,omitempty" bson:"description,omitempty"` // Section description.  Displayed in normal-size text below the label
	Fields      []Field `json:"fields,omitempty"      bson:"fields"`                // Fields included in the Section
}

// Field represents an individual element in a Form.
type Field struct {
	Type        string   `json:"type"                  bson:"type"`                  // The data type of the form item.
	Name        string   `json:"name"                  bson:"name"`                  // The name of the item that's used internally
	Label       string   `json:"label"                 bson:"label"`                 // The label of the item that's displayed to the user
	Options     []string `json:"options,omitempty"     bson:"options"`               // A slice of options to be displayed for certain data types
	Required    bool     `json:"required,omitempty"    bson:"required,omitempty"`    // If TRUE, then this item must be present for the form to be submitted
	Default     string   `json:"default,omitempty"     bson:"default,omitempty"`     // Default value to be set if none is present
	Placeholder string   `json:"placeholder,omitempty" bson:"placeholder,omitempty"` // Placeholder Text to display when the field is empty.
	Hint        string   `json:"hint,omitempty"        bson:"hint,omitempty"`        // Additional, text that may be displayed next to the input widget.
	Public      bool     `json:"public,omitempty"      bson:"public,omitempty"`      // If TRUE, then this field can be edited on the public site, else it cannot
	Exclusions  []string `json:"exclusions,omitempty"  bson:"exclusions,omitempty"`  // List of the items to be excluded
}

// FieldList represents a list of fields that are displayed in a Form
type FieldList []Field

// ParseFieldList tries to convert an interface{} (really []map[string]interface{}) into
// a properly formatted FieldList.  It returns an error if it is unsuccessful.
func ParseFieldList(data interface{}) (FieldList, error) {

	var fieldList FieldList

	if data, ok := data.([]interface{}); ok {

		for _, field := range data {

			if field, ok := field.(map[string]interface{}); ok {

				if field, err := ParseField(field); err == nil {
					fieldList = append(fieldList, field)
				}
			}
		}
	}

	return fieldList, nil
}

// ParseField tries to convert a map[string]interface{} into a
// properly formatted Field record.  It returns an error if it is
// unsuccessful
func ParseField(data map[string]interface{}) (Field, error) {

	var field Field

	if name, ok := data["name"].(string); ok {
		field.Name = name
	}

	if typ, ok := data["type"].(string); ok {
		field.Type = typ
	}

	if label, ok := data["label"].(string); ok {
		field.Label = label
	}

	if options, ok := data["options"].([]interface{}); ok {
		var value []string

		for _, option := range options {
			if option, ok := option.(string); ok {
				value = append(value, option)
			}
		}
		field.Options = value
	}

	if required, ok := data["required"].(bool); ok {
		field.Required = required
	}

	if public, ok := data["public"].(bool); ok {
		field.Public = public
	}

	if field.Name == "" {
		// field.Name = token.New(field.Label)
	}

	return field, nil
}
