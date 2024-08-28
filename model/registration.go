package model

import (
	"html/template"
	"io/fs"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
)

// Registration represents an HTML registration used for building Streams
type Registration struct {
	RegistrationID string               `json:"registrationId"     bson:"registrationId"` // Internal name/token other objects (like streams) will use to reference this Registration.
	Extends        sliceof.String       `json:"extends"            bson:"extends"`        // List of registrations that this registration extends.  The first registration in the list is the most important, and the last registration in the list is the least important.
	Label          string               `json:"label"              bson:"label"`          // Human-readable label used in management UI.
	Description    string               `json:"description"        bson:"description"`    // Human-readable long-description text used in management UI.
	Icon           string               `json:"icon"               bson:"icon"`           // Icon image used in management UI.
	Sort           int                  `json:"sort"               bson:"sort"`           // Sort order used in management UI.
	Form           form.Element         `json:"form"               bson:"form"`           // Form used to edit custom data
	Schema         schema.Schema        `json:"schema"             bson:"schema"`         // JSON Schema that describes the data required to populate this Registration.
	Actions        mapof.Object[Action] `json:"actions"            bson:"actions"`        // Map of actions that can be performed on streams of this Registration
	HTMLTemplate   *template.Template   `json:"-"                  bson:"-"`              // Compiled HTML template
	Bundles        mapof.Object[Bundle] `json:"bundles"            bson:"bundles"`        // Additional resources (JS, HS, CSS) reqired tp remder this Registration.
	Resources      fs.FS                `json:"-"                  bson:"-"`              // File system containing the registration resources
	AllowedFields  []string             `json:"allowedFields"      bson:"allowedFields"`  // List of fields that are allowed to be set by the user
}

// NewRegistration creates a new, fully initialized Registration object
func NewRegistration(registrationID string, funcMap template.FuncMap) Registration {

	return Registration{
		RegistrationID: registrationID,
		Extends:        make([]string, 0),
		Form:           form.NewElement(),
		Actions:        make(map[string]Action),
		HTMLTemplate:   template.New("").Funcs(funcMap),
		AllowedFields:  make([]string, 0),
	}
}

// ID implements the set.Value interface
func (registration Registration) ID() string {
	return registration.RegistrationID
}

func (registration Registration) IsZero() bool {
	if registration.RegistrationID != "" {
		return false
	} else if len(registration.Actions) > 0 {
		return false
	}

	return true
}

// Action returns the action object for a specified name
func (registration *Registration) Action(actionID string) (Action, bool) {
	action, ok := registration.Actions[actionID]
	return action, ok
}

func (registration *Registration) Inherit(parent *Registration) {

	// Null check.
	if parent == nil {
		return
	}

	// Inherit schema items from the parent (if not already defined)
	registration.Schema.Inherit(parent.Schema)

	// Inherit Actions from the parent (if not already defined)
	for actionID, action := range parent.Actions {
		if _, ok := registration.Actions[actionID]; !ok {
			registration.Actions[actionID] = action
		}
	}

	// Inherit HTMLTemplates from the parent (if not already defined)
	for _, templateName := range parent.HTMLTemplate.Templates() {
		if registration.HTMLTemplate.Lookup(templateName.Name()) == nil {
			if _, err := registration.HTMLTemplate.AddParseTree(templateName.Name(), templateName.Tree); err != nil {
				derp.Report(derp.Wrap(err, "model.Template.Inherit", "Error adding template", templateName.Name()))
			}
		}
	}
}
