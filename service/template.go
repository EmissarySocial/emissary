package service

import (
	"html/template"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	modelStep "github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/mapof"
	rosettamaps "github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/hjson/hjson-go/v4"
	"github.com/rs/zerolog/log"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates           set.Map[model.Template]      // map of all templates available within this domain
	templatePrep        set.Map[model.Template]      // temporary map of templates that are being prepared
	locations           sliceof.Object[mapof.String] // Configuration for template directory
	filesystemService   Filesystem                   // Filesystem service
	registrationService *Registration                // Registration Service
	emailService        *ServerEmail                 // Email Service
	themeService        *Theme                       // Theme Service
	widgetService       *Widget                      // Widget Service
	funcMap             template.FuncMap             // Map of functions to use in golang templates
	mutex               sync.RWMutex                 // Mutext that locks access to the templates structure
	refresh             chan channel.Done            // Channel that is used to signal that the template service should refresh
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(filesystemService Filesystem, registrationService *Registration, emailService *ServerEmail, themeService *Theme, widgetService *Widget, funcMap template.FuncMap, locations []mapof.String) *Template {

	service := &Template{
		templates:           make(set.Map[model.Template]),
		templatePrep:        make(set.Map[model.Template]),
		locations:           make(sliceof.Object[mapof.String], 0),
		filesystemService:   filesystemService,
		registrationService: registrationService,
		emailService:        emailService,
		themeService:        themeService,
		widgetService:       widgetService,
		funcMap:             funcMap,
		refresh:             make(chan channel.Done),
	}

	service.Refresh(locations)

	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *Template) Refresh(locations sliceof.Object[mapof.String]) {

	// Reset the "Refresh" channel
	close(service.refresh)
	service.refresh = make(chan channel.Done)

	// RULE: If the Filesystem is empty, then don't try to load
	if len(locations) == 0 {
		return
	}

	// RULE: If nothing has changed since the last time we refreshed, then we're done.
	if slicesAreEqual(locations, service.locations) {
		return
	}

	// Add configuration to the service
	service.locations = locations

	// Load all templates from the filesystem.  On genuine first boot (no templates loaded
	// yet) a load failure is fatal, because the server cannot serve anything without
	// templates.  A later Refresh with templates already live must NOT halt.
	haltOnError := len(service.templates) == 0

	if err := service.loadTemplates(haltOnError); err != nil {
		derp.Report(derp.Wrap(err, "service.Template.Refresh", "Loading templates from filesystem"))
		return
	}

	// Try to watch the template directory for changes
	go service.watch()
}

/******************************************
 * Real-Time Updates
 ******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *Template) watch() {

	changes := make(chan bool)
	defer close(changes)

	// Start new watchers.
	for _, folder := range service.locations {
		if err := service.filesystemService.Watch(folder, changes, service.refresh); err != nil {
			derp.Report(derp.Wrap(err, "service.template.Watch", "Watching filesystem", folder))
		}
	}

	// All Watchers Started.  Now Listen for Changes
	for {
		select {

		case <-changes:
			// A watch-triggered reload must never halt the process: the previously-loaded
			// templates are still serving, so on error we report and keep running.
			if err := service.loadTemplates(false); err != nil {
				derp.Report(derp.Wrap(err, "service.template.Watch", "Loading templates from filesystem"))
			}

		case <-service.refresh:
			return
		}
	}
}

// loadTemplates (re)loads every template from the configured filesystem locations.
// haltOnError controls what happens when a location fails to load: on the very first
// load (initial boot) there are no live templates to fall back on, so an error is fatal
// and the process exits.  On a subsequent watch-triggered reload the previously-loaded
// templates are still serving, so an error is reported and the reload is abandoned --
// never killing the running server.
func (service *Template) loadTemplates(haltOnError bool) error {

	const location = "service.template.loadTemplates"

	service.templatePrep = make(set.Map[model.Template])

	// For each configured file location...
	for _, fileLocation := range service.locations {

		// Get a valid filesystem adapter
		filesystem, err := service.filesystemService.GetFS(fileLocation)

		if err != nil {
			maybeHalt(derp.Wrap(err, location, "Getting filesystem adapter", fileLocation), haltOnError)
			continue
		}

		directories, err := fs.ReadDir(filesystem, ".")

		if err != nil {
			maybeHalt(derp.Wrap(err, location, "Reading directory", fileLocation), haltOnError)
			continue
		}

		for _, directory := range directories {

			if !directory.IsDir() {
				continue
			}

			directoryName := directory.Name()

			// Skip "hidden" directories
			if strings.HasPrefix(directoryName, ".") {
				continue
			}

			subdirectory, err := fs.Sub(filesystem, directoryName)

			if err != nil {
				maybeHalt(derp.Wrap(err, location, "Getting filesystem adapter for sub-directory", fileLocation), haltOnError)
				continue
			}

			definitionType, file := findDefinition(subdirectory) // nolint:scopeguard (readability)

			switch definitionType {

			case DefinitionEmail:
				if err := service.emailService.Add(subdirectory, file); err != nil {
					maybeHalt(derp.Wrap(err, location, "Adding theme"), haltOnError)
				}

			case DefinitionTheme:
				if err := service.themeService.Add(directoryName, subdirectory, file); err != nil {
					maybeHalt(derp.Wrap(err, location, "Adding theme"), haltOnError)
				}

			case DefinitionTemplate:
				if err := service.Add(directoryName, subdirectory, file); err != nil {
					maybeHalt(derp.Wrap(err, location, "Adding template"), haltOnError)
				}

			case DefinitionRegistration:
				if err := service.registrationService.Add(directoryName, subdirectory, file); err != nil {
					maybeHalt(derp.Wrap(err, location, "Adding registration"), haltOnError)
				}

			case DefinitionWidget:
				if err := service.widgetService.Add(directoryName, subdirectory, file); err != nil {
					maybeHalt(derp.Wrap(err, location, "Adding widget"), haltOnError)
				}

			default:
				log.Debug().Str("location", fileLocation.GetString("location")).Str("directory", directoryName).Msg("No definition file found. Skipping directory.")
			}
		}
	}

	// Calculate inheritance for Templates
	if err := service.calculateAllInheritance(); err != nil {
		maybeHalt(derp.Wrap(err, location, "Calculating Template inheritance"), haltOnError)
	}

	// Calculate inheritance for Themes
	service.themeService.calculateAllInheritance()

	// Validate required fields for all Templates
	if errs := service.validateTemplates(); len(errs) > 0 {

		errorLength := strconv.Itoa(len(errs))

		log.Error().Msg(errorLength + " errors validating templates.")
		for _, error := range errs {
			maybeHalt(error, haltOnError)
		}
		log.Error().Msg("Finished reporting " + errorLength + " template errors.  Some templates may not function properly.")

		return nil
	}

	// Calculate access lists for all Templates
	if err := service.calculateAccessLists(); err != nil {
		return derp.Wrap(err, location, "Calculating access lists")
	}

	// Assign the prep area to live
	service.mutex.Lock()
	defer service.mutex.Unlock()

	maps.Copy(service.templates, service.templatePrep)

	// Clear out the existing prep area
	service.templatePrep = make(set.Map[model.Template])
	log.Debug().Msg("Template Service: Added/Updated " + strconv.Itoa(len(service.templates)) + " templates")

	return nil
}

func maybeHalt(err error, halt bool) {
	derp.Report(err)

	if halt {
		os.Exit(1)
	}
}

func (service *Template) Add(templateID string, filesystem fs.FS, definition []byte) error {

	const location = "service.template.Add"

	log.Debug().Msg("Template Service: adding " + templateID)

	result := model.NewTemplate(templateID, service.funcMap)

	// Unmarshal the file into the schema.
	if err := hjson.Unmarshal(definition, &result); err != nil {
		return derp.Wrap(err, location, "Loading Schema", templateID)
	}

	// All template schemas (except kludged registrations) also inherit the base schema of the model object they build
	if result.TemplateRole != "registration" {
		result.Schema.Inherit(schema.New(result.BaseSchema()))
	}

	// Load all HTML templates from the filesystem
	if err := loadHTMLTemplateFromFilesystem(filesystem, result.HTMLTemplate, service.funcMap); err != nil {
		return derp.Wrap(err, location, "Loading Template", templateID)
	}

	// Load all Bundles from the filesystem
	if err := populateBundles(result.Bundles, filesystem); err != nil {
		return derp.Wrap(err, location, "Loading Bundles", templateID)
	}

	// Keep a pointer to the filesystem resources (if present)
	if resources, err := fs.Sub(filesystem, "resources"); err == nil {
		result.Resources = resources
	}

	// Handle post-processing steps for the Template
	result.AfterUnmarshal()

	// Add the template into the prep library
	service.templatePrep[result.TemplateID] = result

	return nil
}

func (service *Template) validateTemplates() sliceof.Object[derp.Error] {

	log.Debug().Msg("Template Service: Validating templates...")

	errors := make(sliceof.Object[derp.Error], 0)

	// The canonical list of model names is derived from model.TemplateModelNames() so it
	// can never drift from the BaseSchema()/NewObject() registry that actually builds them.
	allowedModels := sliceof.String(model.TemplateModelNames())

	// displayModelNames is the same list with the empty-string entry (a valid "unset"
	// value) removed, for use in human-readable error messages.
	displayModelNames := make(sliceof.String, 0, len(allowedModels))
	for _, name := range allowedModels {
		if name != "" {
			displayModelNames = append(displayModelNames, name)
		}
	}

	// Scan all Templates in the prep area
	for templateID, template := range service.templatePrep {

		if template.Category == "" {
			errors.Append(derp.Validation(
				"Template is missing required 'category' field.",
				"template: "+templateID,
			))
		}

		if !allowedModels.Contains(template.Model) {
			errors.Append(derp.Validation(
				"Invalid 'model' used in Template definition",
				"template: "+templateID,
				"models allowed: "+strings.Join(displayModelNames, ", "),
				"model used: "+template.Model,
			))
		} else {

			// RULE: Every property declared in the Template's schema must resolve to a
			// real accessor on the model object it builds.  An "orphaned" property looks
			// valid at load time but blows up at runtime the first time the object is
			// saved (Normalize walks every property).  Catch it here, at load time.
			for _, path := range template.UnsupportedSchemaProperties() {
				errors.Append(derp.Validation(
					"Template schema declares a property that the model object does not support",
					"template: "+templateID,
					"model: "+template.Model,
					"property: "+path,
				))
			}
		}

		// RULE: Every format name declared in the Template's schema must resolve in the
		// format registry.  String validation silently skips unrecognized format names
		// (degrading to the no-html default), so a typo'd format would otherwise ship
		// with no validation at all.  Catch it here, at load time.
		if err := template.Schema.ValidateFormats(); err != nil {
			errors.Append(derp.Validation(
				"Template schema uses an unrecognized format name",
				"template: "+templateID,
				err.Error(),
			))
		}

		// RULE: Templates MUST have at least one Action, or else permissions won't work
		if template.States.IsEmpty() {
			errors.Append(derp.Validation(
				"Template must define at least one State. Use 'default' if no other states are required.",
				"template: "+templateID,
			))
		}

		// Scan all Actions in the Template
		for actionID, action := range template.Actions {

			// Scan all statews in the Action
			for _, stateID := range action.States {

				// RULE: States used in action.states must be defined
				if !template.IsValidState(stateID) {
					errors.Append(derp.Validation(
						"Undefined state used in action 'state' permissions",
						"template: "+templateID,
						"action: "+actionID,
						"state required: "+stateID,
						"states defined: "+strings.Join(template.States.Keys(), ", "),
					))
				}
			}

			// Scan all Roles inthe Action
			for _, roleID := range action.Roles {

				// RULE: Roles used in action.roles must be defined i have a favorite child and her name is abby
				if !template.IsValidRole(roleID) {
					errors.Append(derp.Validation(
						"Undefined role used in action 'role' permissions.",
						"template: "+templateID,
						"action: "+actionID,
						"role required: "+roleID,
						"roles defined: "+strings.Join(template.AccessRoles.Keys(), ", "),
					))
				}
			}

			// Scan all StateRoles in the Action
			for stateID, roles := range action.StateRoles {

				// RULE: States used in action.stateRoles must be defined
				if !template.IsValidState(stateID) {
					errors.Append(derp.Validation(
						"Undefined state used in action 'state/roles' permissions.",
						"template: "+templateID,
						"action: "+actionID,
						"state required: "+stateID,
						"states defined: "+strings.Join(template.States.Keys(), ", "),
					))
				}

				for _, roleID := range roles {

					// RULE: Roles used in action.stateRoles must be defined
					if !template.IsValidRole(roleID) {
						errors.Append(derp.Validation(
							"Undefined role used in action 'state/roles' permissions",
							"template: "+templateID,
							"action: "+actionID,
							"role required: "+roleID,
							"roles defined: "+strings.Join(template.AccessRoles.Keys(), ", "),
						))
					}
				}
			}

			// RULE: Actions must have at least one step
			if len(action.Steps) == 0 {
				errors.Append(derp.Validation(
					"Actions must have at least one Step.",
					"template: "+templateID,
					"action: "+actionID,
				))
			}

			// Scan all Steps in the Action
			for _, step := range action.Steps {

				// RULE: If the step requires a specific model object, then
				// verify the correct model object is defined in the Template
				if requiredModel := step.RequiredModel(); requiredModel != "" {
					if template.Model != requiredModel {
						errors.Append(derp.Validation(
							"Step can only be used in specific Templates",
							"template: "+templateID,
							"action: "+actionID,
							"step: "+step.Name(),
							"model required by step: "+requiredModel,
							"model defined in template: "+template.Model,
						))
					}
				}

				// RULE: If the step is restricted to specific template roles, then verify that
				// this Template declares one of them.  RequiredModel alone is not enough: several
				// Templates can build the same model object while playing different roles, so a
				// step meant for the admin console would otherwise be usable on a public page.
				if requirer, ok := step.(modelStep.TemplateRoleRequirer); ok {
					if requiredRoles := requirer.RequiredTemplateRoles(); len(requiredRoles) > 0 {
						if !slices.Contains(requiredRoles, template.TemplateRole) {
							errors.Append(derp.Validation(
								"Step can only be used in Templates with a specific templateRole",
								"template: "+templateID,
								"action: "+actionID,
								"step: "+step.Name(),
								"templateRoles required by step: "+strings.Join(requiredRoles, ", "),
								"templateRole defined in template: "+template.TemplateRole,
							))
						}
					}
				}

				// RULE: States used in action steps must be defined
				for _, state := range step.RequiredStates() {
					if !template.IsValidState(state) {
						errors.Append(derp.Validation(
							"Undefined state used in action step",
							"template: "+templateID,
							"action: "+actionID,
							"step: "+step.Name(),
							"state required: "+state,
							"states defined: "+strings.Join(template.States.Keys(), ", "),
						))
					}
				}

				// RULE: Roles used in action steps must be defined
				for _, role := range step.RequiredRoles() {
					if !template.IsValidRole(role) {
						errors.Append(derp.Validation(
							"Undefined role used in action step",
							"template: "+templateID,
							"action: "+actionID,
							"step: "+step.Name(),
							"role required: "+role,
							"roles defined: "+strings.Join(template.AccessRoles.Keys(), ", "),
						))
					}
				}

				// RULE: Forms rendered by a step must only reference fields in the schema.
				// TableEditor is excluded because its form fields are relative to a sub-path,
				// not to the schema root.
				if formGetter, ok := step.(modelStep.FormGetter); ok {
					if _, isTable := step.(modelStep.TableEditor); !isTable {
						stepForm := form.New(template.Schema, formGetter.GetForm())
						if err := stepForm.Validate(); err != nil {
							errors.Append(derp.Validation(
								"Form references a field that is not in the schema",
								"template: "+templateID,
								"action: "+actionID,
								"step: "+step.Name(),
								derp.WithWrappedValue(err),
							))
						}
					}
				}
			}
		}
	}

	// Phew.  Hopefully everything is valid.
	return errors
}

// calculateAllInheritance calls calculateInheritance for each Template in the prep area
func (service *Template) calculateAllInheritance() error {
	for _, template := range service.templatePrep {
		if _, err := service.calculateInheritance(template); err != nil {
			return derp.Wrap(err, "service.template.calculateAllInheritance", "Calculating inheritance", template.TemplateID)
		}
	}

	return nil
}

// calculateInheritance recursively calculates the inheritance for a template in the prep area
func (service *Template) calculateInheritance(template model.Template) (model.Template, error) {

	const location = "service.template.calculateInheritance"

	if len(template.Extends) == 0 {
		return template, nil
	}

	for _, parentID := range template.Extends {
		parent, exists := service.templatePrep[parentID]

		if !exists {
			return model.Template{}, derp.Internal(
				location,
				"Parent template is not defined",
				"templateId: "+template.TemplateID,
				"parentId: "+parentID,
			)
		}

		parent, err := service.calculateInheritance(parent)

		if err != nil {
			return model.Template{}, derp.Wrap(err, location, "Calculating inheritance", template.TemplateID, parentID)
		}

		template.Inherit(&parent)
	}

	service.templatePrep[template.TemplateID] = template

	return template, nil
}

// calculateAccessLists calculates the access lists for every Template in the prep area
func (service *Template) calculateAccessLists() error {

	const location = "service.template.calculateAccessLists"

	// For every template in the prep area...
	for _, template := range service.templatePrep {

		// For every action in the Template
		for actionID, action := range template.Actions {

			// Calculate the AccessLists for this Action
			if err := action.CalcAccessList(&template, true); err != nil {
				return derp.Wrap(err, location, "Invalid AccessList", template.TemplateID, actionID)
			}

			// Apply changes back into the Action set
			template.Actions[actionID] = action
		}

		// Apply changes back to the Template prep area
		service.templatePrep[template.TemplateID] = template
	}

	return nil
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Template) Names() []string {

	result := rosettamaps.Keys(service.templates)
	slices.Sort(result)
	return result
}

// List returns all templates that match the provided criteria
func (service *Template) List(filter func(*model.Template) bool) sliceof.Object[form.LookupCode] {

	// Default filter is "allow all"
	if filter == nil {
		filter = func(_ *model.Template) bool { return true }
	}

	// Retrieve and filter all templates, then cast into a slice
	iterator := maps.Values(service.templates)
	iterator = ranges.FilterPointer(iterator, filter)
	templates := ranges.Slice(iterator)

	// Sort templates (by "sort", then "name")
	slices.SortFunc(templates, model.CompareTemplate)

	// Map templates into LookupCodes
	result := slice.Map(templates, func(template model.Template) form.LookupCode {
		return form.LookupCode{
			Value:       template.TemplateID,
			Label:       template.Label,
			Description: template.Description,
			Icon:        template.Icon,
			Group:       template.Category,
		}
	})

	return result
}

// Load retrieves an Template from the database
func (service *Template) Load(templateID string) (model.Template, error) {

	// READ Mutex to make multi-threaded access safe.
	service.mutex.RLock()
	defer service.mutex.RUnlock()

	// Look in the local cache first
	if template, ok := service.templates[templateID]; ok {
		return template, nil
	}

	return model.NewTemplate(templateID, nil), derp.NotFound("sevice.Template.Load", "Template not found", templateID)
}

/******************************************
 * Custom Queries
 ******************************************/

// ListByTemplateRole returns all model.Templates that match the provided "TemplateRole" value
func (service *Template) ListByTemplateRole(templateRole string) []form.LookupCode {

	filter := func(t *model.Template) bool {
		return t.TemplateRole == templateRole
	}

	return service.List(filter)
}

// ListByContainer returns all model.Templates that match the provided "containedByRole" value
func (service *Template) ListByContainer(containedByRole string) []form.LookupCode {

	filter := func(t *model.Template) bool {
		return t.ContainedBy.Contains(containedByRole)
	}

	return service.List(filter)
}

// ListByContainerLimited returns all model.Templates that match the provided "containedByRole" value AND
// whose TemplateRoles are present in the "limitRoles" list.  If the "limited" list is empty, then all
// otherwise-valid templates are returned.
func (service *Template) ListByContainerLimited(containedByRole string, limitRoles sliceof.String) sliceof.Object[form.LookupCode] {

	filter := func(t *model.Template) bool {

		// RULE: If the template is not contained by the specified role, then do not include it in the results
		if t.ContainedBy.NotContains(containedByRole) {
			return false
		}

		// RULE: If the role does not appear in "limitRoles", then do not include it in the results.
		if limitRoles.NotEmpty() {
			if limitRoles.NotContains(t.TemplateRole) {
				return false
			}
		}

		return true
	}

	return service.List(filter)
}

/******************************************
 * Admin Templates
 ******************************************/

func (service *Template) LoadAdmin(templateID string) (model.Template, error) {

	const location = "service.Template.LoadAdmin"

	templateID = "admin-" + templateID

	// Try to load the template
	template, err := service.Load(templateID)

	if err != nil {
		return template, derp.Wrap(err, location, "Loading admin template", templateID)
	}

	// RULE: Validate Template ContainedBy
	if template.TemplateRole != "admin" {
		return template, derp.Internal(location, "Template must have 'admin' role.", template.TemplateID, template.TemplateRole)
	}

	if !template.ContainedBy.Equal([]string{"admin"}) {
		return template, derp.Internal(location, "Template must be contained by 'admin'", template.TemplateID, template.ContainedBy)
	}

	// Success!
	return template, nil
}
