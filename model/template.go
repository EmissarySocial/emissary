package model

import (
	"html/template"
	"io/fs"
	"slices"
	"strings"

	"github.com/EmissarySocial/emissary/tools/templatemap"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/compare"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/rosetta/translate"
)

// Template represents an HTML template used for building Streams
type Template struct {
	TemplateID         string               `json:"templateId"         bson:"templateId"`         // Internal name/token other objects (like streams) will use to reference this Template.
	URL                string               `json:"url"                bson:"url"`                // URL where this template is published
	TemplateRole       string               `json:"templateRole"       bson:"templateRole"`       // Role that this Template performs in the system.  Used to match which streams can be contained by which other streams.
	SocialRole         string               `json:"socialRole"         bson:"socialRole"`         // Role to use for this Template in social integrations (Article, Note, etc)
	SocialRules        translate.Pipeline   `json:"socialRules"        bson:"socialRules"`        // List of rules to convert this Template into a social object
	Model              string               `json:"model"              bson:"model"`              // Type of model object that this template works with. (Stream, User, Group, Domain, etc.)
	Extends            sliceof.String       `json:"extends"            bson:"extends"`            // List of templates that this template extends.  The first template in the list is the most important, and the last template in the list is the least important.
	ContainedBy        sliceof.String       `json:"containedBy"        bson:"containedBy"`        // Slice of Templates that can contain Streams that use this Template.
	Label              string               `json:"label"              bson:"label"`              // Human-readable label used in management UI.
	Description        string               `json:"description"        bson:"description"`        // Human-readable long-description text used in management UI.
	Category           string               `json:"category"           bson:"category"`           // Human-readable category (grouping) used in management UI.
	Icon               string               `json:"icon"               bson:"icon"`               // Icon image used in management UI.
	Sort               int                  `json:"sort"               bson:"sort"`               // Sort order used in management UI.
	ChildSortType      string               `json:"childSortType"      bson:"childSortType"`      // SortType used to display children
	ChildSortDirection string               `json:"childSortDirection" bson:"childSortDirection"` // Sort direction "asc" or "desc" (Default is ascending)
	WidgetLocations    sliceof.String       `json:"widget-locations"   bson:"widgetLocations"`    // List of locations where widgets can be placed.  Common values are: "TOP", "BOTTOM", "LEFT", "RIGHT"
	Schema             schema.Schema        `json:"schema"             bson:"schema"`             // JSON Schema that describes the data required to populate this Template.
	States             mapof.Object[State]  `json:"states"             bson:"states"`             // Map of States (by state.ID) that Streams of this Template can be in.
	AccessRoles        mapof.Object[Role]   `json:"roles"              bson:"accessRoles"`        // Map of custom roles defined by this Template.
	Actions            mapof.Object[Action] `json:"actions"            bson:"actions"`            // Map of actions that can be performed on streams of this Template
	HTMLTemplate       *template.Template   `json:"-"                  bson:"-"`                  // Compiled HTML template
	TagPaths           []string             `json:"tagPaths"           bson:"tagPaths"`           // List of schema paths whose values are scanned for #hashtags
	TagURL             string               `json:"tagUrl"             bson:"tagUrl"`             // URL prefix for hashtag links rendered into content ("%23" + tag is appended)
	SearchOptions      templatemap.Map      `json:"search"             bson:"search"`             // Compiled templates that override default search result values
	Bundles            mapof.Object[Bundle] `json:"bundles"            bson:"bundles"`            // Additional resources (JS, HS, CSS) reqired tp remder this Template.
	Resources          fs.FS                `json:"-"                  bson:"-"`                  // File system containing the template resources
	Datasets           DatasetMap           `json:"datasets"           bson:"-"`                  // Lookup codes defined by this template
	DefaultAction      string               `json:"defaultAction"      bson:"defaultAction"`      // Name of the action to be used when none is provided.  Also serves as the permissions for viewing a Stream.  If this is empty, it is assumed to be "view"
	Actor              StreamActor          `json:"actor"              bson:"actor"`              // ActivityPub Actor operated on behalf of this Template/Stream
}

type DatasetMap map[string]form.ReadOnlyLookupGroup

// NewTemplate creates a new, fully initialized Template object
func NewTemplate(templateID string, funcMap template.FuncMap) Template {

	return Template{
		TemplateID:         templateID,
		SocialRules:        make(translate.Pipeline, 0),
		Extends:            make([]string, 0),
		ContainedBy:        make([]string, 0),
		ChildSortType:      "rank",
		ChildSortDirection: option.SortDirectionAscending,
		WidgetLocations:    make(sliceof.String, 0),
		States:             make(map[string]State),
		AccessRoles:        make(map[string]Role),
		Actions:            make(map[string]Action),
		DefaultAction:      "view",
		HTMLTemplate:       template.New("").Funcs(funcMap),
		SearchOptions:      make(templatemap.Map),
		Bundles:            make(map[string]Bundle),
		Datasets:           make(DatasetMap),
	}
}

// ID implements the set.Value interface
func (template Template) ID() string {
	return template.TemplateID
}

// AfterUnmarshal performs some post-processing on the Template object
// after it has been unmarshalled from JSON.
func (template *Template) AfterUnmarshal() {

	// Apply RoleIDs to each AccessRole
	for roleID, accessRole := range template.AccessRoles {
		accessRole.RoleID = roleID
		template.AccessRoles[roleID] = accessRole
	}
}

// IsZero returns TRUE if this Template is a zero value
func (template Template) IsZero() bool {

	if template.TemplateID != "" {
		return false
	}

	if template.TemplateRole != "" {
		return false
	}

	if len(template.Actions) > 0 {
		return false
	}

	return true
}

// CanBeContainedBy returns TRUE if this Streams using this Template can be nested inside of
// Streams using the Template named in the parameters
func (template *Template) CanBeContainedBy(templateRoles ...string) bool {

	// Otherwise, this template MUSt list the potential parent Stream's *role* in its ContainedBy list
	for _, templateRole := range templateRoles {
		if slice.Contains(template.ContainedBy, templateRole) {
			return true
		}
	}
	return false
}

func (template *Template) IsValidWidgetLocation(location string) bool {

	// NILCHECK: Template cannot be nil
	if template == nil {
		return false
	}

	return slice.Contains(template.WidgetLocations, location)
}

// IsValidRole returns TRUE if the provided roleID is valid for this Template
func (template *Template) IsValidRole(roleID string) bool {

	switch roleID {

	// MagicRoles are always valid
	case MagicRoleOwner:
	case MagicRoleAnonymous:
	case MagicRoleAuthenticated:
	case MagicRoleMyself:
	case MagicRoleAuthor:

	// Custom roles must be defined in the Template
	default:
		if _, ok := template.AccessRoles[roleID]; !ok {
			return false
		}
	}

	return true
}

// IsValidState returns TRUE if the provided stateID is valid for this Template
func (template *Template) IsValidState(stateID string) bool {

	// NILCHECK: Template cannot be nil
	if template == nil {
		return false
	}

	if _, ok := template.States[stateID]; !ok {
		return false
	}
	return true
}

// State searches for the State in this Template that matches the provided StateID
// If found, it is returned along with a TRUE
// If not found, an empty state is returned along with a FALSE
func (template *Template) State(stateID string) (State, bool) {

	// NILCHECK: Template cannot be nil
	if template == nil {
		return State{}, false
	}

	state, ok := template.States[stateID]
	return state, ok
}

// Action returns the action object for a specified name
func (template *Template) Action(actionID string) (Action, bool) {

	// NILCHECK: Template cannot be nil
	if template == nil {
		return Action{}, false
	}

	action, ok := template.Actions[actionID]
	return action, ok
}

// Default returns the default Action for this Template.
func (template *Template) Default() Action {

	// NILCHECK: Template cannot be nil
	if template == nil {
		return Action{}
	}

	return template.Actions[template.DefaultAction]
}

func (template *Template) Inherit(parent *Template) {

	// NILCHECK: Parent cannot be nil
	if parent == nil {
		return
	}

	// Inherit schema items from the parent.
	template.Schema.Inherit(parent.Schema)

	// Inherit WidgetLocations.
	if len(template.WidgetLocations) == 0 {
		template.WidgetLocations = parent.WidgetLocations
	}

	// Inherit TemplateRole.
	if template.TemplateRole == "" {
		template.TemplateRole = parent.TemplateRole
	}

	// Inherit SocialRole.
	if template.SocialRole == "" {
		template.SocialRole = parent.SocialRole
	}

	// Inherit ContainedBy.
	if len(template.ContainedBy) == 0 {
		template.ContainedBy = parent.ContainedBy
	}

	// Inherit Model.
	if template.Model == "" {
		template.Model = parent.Model
	}

	// Inherit TagURL. (TagPaths deliberately does NOT inherit; external template packages depend on that.)
	if template.TagURL == "" {
		template.TagURL = parent.TagURL
	}

	// Apply SocialRules
	if len(parent.SocialRules) > 0 {
		template.SocialRules = append(parent.SocialRules, template.SocialRules...)
	}

	// Inherit Datasets from the parent
	for datasetID, dataset := range parent.Datasets {
		if _, exists := template.Datasets[datasetID]; !exists {
			template.Datasets[datasetID] = dataset
		}
	}

	// Inherit SearchTemplate
	for optionID, option := range parent.SearchOptions {
		if _, exists := template.SearchOptions[optionID]; !exists {
			template.SearchOptions[optionID] = option
		}
	}

	// Inherit Roles from the parent.
	for roleID, role := range parent.AccessRoles {
		if _, exists := template.AccessRoles[roleID]; !exists {
			template.AccessRoles[roleID] = role
		}
	}

	// Inherit States from the parent.
	for stateID, state := range parent.States {
		if _, exists := template.States[stateID]; !exists {
			template.States[stateID] = state
		}
	}

	// Inherit Actions from the parent.
	for actionID, action := range parent.Actions {
		if _, exists := template.Actions[actionID]; !exists {
			template.Actions[actionID] = action
		}
	}

	// Inherit HTMLTemplates from the parent.
	for _, templateName := range parent.HTMLTemplate.Templates() {
		if template.HTMLTemplate.Lookup(templateName.Name()) == nil {
			if _, err := template.HTMLTemplate.AddParseTree(templateName.Name(), templateName.Tree); err != nil {
				derp.Report(derp.Wrap(err, "model.Template.Inherit", "Adding template", templateName.Name()))
			}
		}
	}
}

// IsSearch returns TRUE if this is Template is a search engine
func (template Template) IsSearch() bool {
	return template.TemplateRole == "search"
}

// IsSubscribable returns TRUE if this Template has at-least-one role that requires a product
func (template Template) IsSubscribable() bool {

	for _, role := range template.AccessRoles {
		if role.IsPrivileged {
			return true
		}
	}

	return false
}

func (template Template) PrivilegedRoles() []Role {

	result := make([]Role, 0)

	for _, role := range template.AccessRoles {
		if role.IsPrivileged {
			result = append(result, role)
		}
	}

	// Sort the results by the role Label
	slices.SortFunc(result, func(a, b Role) int {
		return compare.String(a.Label, b.Label)
	})

	return result
}

/******************************************
 * OEmbed Methods
 ******************************************/

// TODO: (oembed/TODO.md Phases 9+11) These two lookups have no callers anywhere.
// Re-evaluate when the oEmbed rework lands: either wire them into the /.oembed
// handler so template packages can serve custom (rich/video) oEmbed documents
// via the Phase 9 primitives, or delete them. Consumer-side metadata is moving
// to sherlock/metadata and will not use these.

// HasOEmbed returns TRUE if this Template has an OEmbed template
func (template Template) HasOEmbed() bool {
	return template.HTMLTemplate.Lookup("oembed") != nil
}

// GetOEmbed returns the OEmbed template for this Template
// If no OEmbed template is found, then nil is returned
func (template Template) GetOEmbed() *template.Template {
	return template.HTMLTemplate.Lookup("oembed")
}

// templateModel pairs the schema and the object constructor for a single model type.
// Keeping them together in ONE registry entry guarantees BaseSchema() and NewObject()
// can never disagree about which schema validates which object.
type templateModel struct {
	schema    func() schema.Element // baseline schema for this model's builder
	newObject func() any            // fresh, zero-value instance of this model's object
}

// templateModelRegistry is the single source of truth mapping a Template's "Model"
// property to the schema + object its builder uses.  This mapping mirrors the builder
// dispatch in handler/admin.go and each builder's schema() method, so that load-time
// validation of a Template's forms uses the same schema the forms will see at runtime.
//
// Model names NOT listed here (including "Stream", "None", and the empty string) fall
// back to the Stream model via templateModelForName().  Every list that used to
// enumerate model names by hand -- BaseSchema, NewObject, the load-time allowedModels
// whitelist, and the drift test -- now derives from this one map.
var templateModelRegistry = map[string]templateModel{
	// The Outbox, Inbox, Settings, User, Conversations, and Notifications builders all build User objects
	"User":          {schema: UserSchema, newObject: newObjectPointer(NewUser)},
	"Outbox":        {schema: UserSchema, newObject: newObjectPointer(NewUser)},
	"Inbox":         {schema: UserSchema, newObject: newObjectPointer(NewUser)},
	"Settings":      {schema: UserSchema, newObject: newObjectPointer(NewUser)},
	"Conversations": {schema: UserSchema, newObject: newObjectPointer(NewUser)},
	"Notifications": {schema: UserSchema, newObject: newObjectPointer(NewUser)},

	// The Domain, Search, SSO, Followers, Following, and Syndication builders all build Domain objects
	"Domain":      {schema: DomainSchema, newObject: newObjectPointer(NewDomain)},
	"Search":      {schema: DomainSchema, newObject: newObjectPointer(NewDomain)},
	"SSO":         {schema: DomainSchema, newObject: newObjectPointer(NewDomain)},
	"Followers":   {schema: DomainSchema, newObject: newObjectPointer(NewDomain)},
	"Following":   {schema: DomainSchema, newObject: newObjectPointer(NewDomain)},
	"Syndication": {schema: DomainSchema, newObject: newObjectPointer(NewDomain)},

	"Group":    {schema: GroupSchema, newObject: newObjectPointer(NewGroup)},
	"Identity": {schema: IdentitySchema, newObject: newObjectPointer(NewIdentity)},
	"Rule":     {schema: RuleSchema, newObject: newObjectPointer(NewRule)},
	"Tag":      {schema: SearchTagSchema, newObject: newObjectPointer(NewSearchTag)},
	"Webhook":  {schema: WebhookSchema, newObject: newObjectPointer(NewWebhook)},
}

// templateModelStreamDefault is the entry used for "Stream", "None", the empty string,
// and any model name not present in templateModelRegistry.
var templateModelStreamDefault = templateModel{schema: StreamSchema, newObject: newObjectPointer(NewStream)}

// newObjectPointer adapts a value constructor (e.g. NewUser) into a func that returns
// a pointer to a fresh instance, as required by the schema getter/setter interfaces.
func newObjectPointer[T any](constructor func() T) func() any {
	return func() any {
		value := constructor()
		return &value
	}
}

// templateModelForName resolves a model name to its registry entry, falling back to the
// Stream default for "Stream", "None", "", and any unrecognized name.
func templateModelForName(model string) templateModel {
	if entry, found := templateModelRegistry[model]; found {
		return entry
	}
	return templateModelStreamDefault
}

// TemplateModelNames returns every model name the system recognizes: the explicit
// registry keys plus the names that implicitly resolve to the Stream model.  This is the
// canonical whitelist -- callers (e.g. load-time Template validation) should use it
// instead of maintaining their own list.
func TemplateModelNames() []string {
	result := make([]string, 0, len(templateModelRegistry)+3)

	for name := range templateModelRegistry {
		result = append(result, name)
	}

	// Names that implicitly resolve to the Stream model (the registry's default).
	result = append(result, "Stream", "None", "")

	return result
}

// BaseSchema returns the schema of the model object that this Template builds,
// as identified by this Template's "Model" property.
func (template Template) BaseSchema() schema.Element {
	return templateModelForName(template.Model).schema()
}

// NewObject returns a fresh, zero-value instance of the model object that this
// Template builds, as identified by this Template's "Model" property.  Because both
// BaseSchema() and NewObject() read the SAME registry entry, the schema and the object
// it validates are guaranteed to stay in lockstep.
func (template Template) NewObject() any {
	return templateModelForName(template.Model).newObject()
}

// UnsupportedSchemaProperties returns the property paths in this Template's schema that do NOT
// resolve to a real accessor on the model object the Template builds
func (template Template) UnsupportedSchemaProperties() []string {

	// These "orphans" look valid at load time but blow up the first time the object is saved.  They
	// are detected with the SAME operation Stream.Save runs -- schema.Normalize against a fresh
	// model object -- so the check matches runtime exactly.  ("data.*" properties are NOT orphans:
	// Normalize writes them even though a naive Get would fail on the empty map.)

	// Only Stream templates normalize against their own Template schema at save time.  Every other
	// model saves against a FIXED model schema whose builder ignores the Template schema, so extra
	// properties there (the virtual "new_password" field, widget config) are harmless.
	if template.Model != "Stream" {
		return nil
	}

	// Fast path: Normalize the whole object exactly as Stream.Save does.  If it succeeds,
	// there are no orphaned properties and we can return immediately.
	if _, err := template.Schema.Normalize(template.NewObject()); err == nil {
		return nil
	}

	// Something failed to normalize.  Localize the offender(s) by normalizing each leaf
	// property in isolation against a fresh object, and collect the ones whose failure is
	// the getter's "unsupported property" signature (as opposed to a value/validation
	// error, which is not an orphan-schema problem).
	result := make([]string, 0)

	for path, element := range template.Schema.AllProperties() {

		propertySchema := schema.New(pathToSchema(path, element))

		if _, err := propertySchema.Normalize(template.NewObject()); err != nil {
			if strings.Contains(derp.Message(derp.RootCause(err)), "does not support this") {
				result = append(result, path)
			}
		}
	}

	slices.Sort(result)
	return result
}

// pathToSchema wraps a single leaf element in the nested Object structure implied by its
// dotted path, so it can be normalized in isolation.  e.g. ("content.type", String{})
// becomes Object{content: Object{type: String{}}}.
func pathToSchema(path string, element schema.Element) schema.Element {

	segments := strings.Split(path, ".")

	// Build from the leaf outward.
	for i := len(segments) - 1; i >= 0; i-- {
		element = schema.Object{Properties: schema.ElementMap{segments[i]: element}}
	}

	return element
}

func CompareTemplate(a, b Template) int {

	if compareSort := compare.Int(a.Sort, b.Sort); compareSort != 0 {
		return compareSort
	}

	return compare.String(a.TemplateID, b.TemplateID)
}
