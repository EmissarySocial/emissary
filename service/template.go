package service

import (
	"html/template"
	"io/fs"
	"maps"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
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

	// Load all templates from the filesystem
	if err := service.loadTemplates(); err != nil {
		derp.Report(derp.Wrap(err, "service.Template.Refresh", "Error loading templates from filesystem"))
		return
	}

	// Try to watch the template directory for changes
	go service.watch()
}

/******************************************
 * REAL-TIME UPDATES
 ******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *Template) watch() {

	changes := make(chan bool)
	defer close(changes)

	// Start new watchers.
	for _, folder := range service.locations {
		if err := service.filesystemService.Watch(folder, changes, service.refresh); err != nil {
			derp.Report(derp.Wrap(err, "service.template.Watch", "Error watching filesystem", folder))
		}
	}

	// All Watchers Started.  Now Listen for Changes
	for {
		select {

		case <-changes:
			if err := service.loadTemplates(); err != nil {
				derp.Report(derp.Wrap(err, "service.template.Watch", "Error loading templates from filesystem"))
			}

		case <-service.refresh:
			return
		}
	}
}

// loadTemplates retrieves the template from the filesystem and parses it into
func (service *Template) loadTemplates() error {

	const location = "service.template.loadTemplates"

	service.templatePrep = make(set.Map[model.Template])

	// For each configured file location...
	for _, fileLocation := range service.locations {

		// Get a valid filesystem adapter
		filesystem, err := service.filesystemService.GetFS(fileLocation)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error getting filesystem adapter", fileLocation))
			continue
		}

		directories, err := fs.ReadDir(filesystem, ".")

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error reading directory", fileLocation))
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
				derp.Report(derp.Wrap(err, location, "Error getting filesystem adapter for sub-directory", fileLocation))
				continue
			}

			definitionType, file, err := findDefinition(subdirectory)

			if err != nil {
				derp.Report(derp.Wrap(err, location, "Invalid definition", fileLocation, directoryName))
				continue
			}

			switch definitionType {

			case DefinitionEmail:
				if err := service.emailService.Add(subdirectory, file); err != nil {
					derp.Report(derp.Wrap(err, location, "Error adding theme"))
				}

			case DefinitionTheme:
				if err := service.themeService.Add(directoryName, subdirectory, file); err != nil {
					derp.Report(derp.Wrap(err, location, "Error adding theme"))
				}

			case DefinitionTemplate:
				if err := service.Add(directoryName, subdirectory, file); err != nil {
					derp.Report(derp.Wrap(err, location, "Error adding template"))
				}

			case DefinitionRegistration:
				if err := service.registrationService.Add(directoryName, subdirectory, file); err != nil {
					derp.Report(derp.Wrap(err, location, "Error adding registration"))
				}

			case DefinitionWidget:
				if err := service.widgetService.Add(directoryName, subdirectory, file); err != nil {
					derp.Report(derp.Wrap(err, location, "Error adding widget"))
				}

			default:
				derp.Report(derp.InternalError(location, "Unrecognized definition type", fileLocation, definitionType))
			}
		}
	}

	// Calculate inheritance for Templates
	if err := service.calculateAllInheritance(); err != nil {
		derp.Report(derp.Wrap(err, location, "Error calculating Template inheritance"))
	}

	// Calculate inheritance for Themes
	service.themeService.calculateAllInheritance()

	// Validate required fields for all Templates
	if errs := service.validateTemplates(); len(errs) > 0 {

		errorLength := strconv.Itoa(len(errs))

		log.Error().Msg(errorLength + " errors validating templates.")
		for _, error := range errs {
			derp.Report(error)
		}
		log.Error().Msg("Finished reporting " + errorLength + " template errors.  Some templates may not function properly.")

		return nil
	}

	// Calculate access lists for all Templates
	if err := service.calculateAllowLists(); err != nil {
		return derp.Wrap(err, location, "Error calculating access lists")
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

func (service *Template) Add(templateID string, filesystem fs.FS, definition []byte) error {

	const location = "service.template.Add"

	log.Debug().Msg("Template Service: adding " + templateID)

	result := model.NewTemplate(templateID, service.funcMap)

	// Unmarshal the file into the schema.
	if err := hjson.Unmarshal(definition, &result); err != nil {
		return derp.Wrap(err, location, "Error loading Schema", templateID)
	}

	// All template schemas (except kludged registrations) also inherit from the main stream schema
	if result.TemplateRole != "registration" {
		result.Schema.Inherit(schema.New(model.StreamSchema()))
	}

	// Load all HTML templates from the filesystem
	if err := loadHTMLTemplateFromFilesystem(filesystem, result.HTMLTemplate, service.funcMap); err != nil {
		return derp.Wrap(err, location, "Error loading Template", templateID)
	}

	// Load all Bundles from the filesystem
	if err := populateBundles(result.Bundles, filesystem); err != nil {
		return derp.Wrap(err, location, "Error loading Bundles", templateID)
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

	// Scan all Templates in the prep area
	for templateID, template := range service.templatePrep {

		// RULE: Templates MUST have at least one Action, or else permissions won't work
		if template.States.IsEmpty() {
			errors.Append(derp.ValidationError(
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
					errors.Append(derp.ValidationError(
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
					errors.Append(derp.ValidationError(
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
					errors.Append(derp.ValidationError(
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
						errors.Append(derp.ValidationError(
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
				errors.Append(derp.ValidationError(
					"Actions must have at least one Step.",
					"template: "+templateID,
					"action: "+actionID,
				))
			}

			// Scan all Steps in the Action
			for _, step := range action.Steps {

				// RULE: States used in action steps must be defined
				for _, state := range step.RequiredStates() {
					if !template.IsValidState(state) {
						errors.Append(derp.ValidationError(
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
						errors.Append(derp.ValidationError(
							"Undefined role used in action step",
							"template: "+templateID,
							"action: "+actionID,
							"step: "+step.Name(),
							"role required: "+role,
							"roles defined: "+strings.Join(template.AccessRoles.Keys(), ", "),
						))
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
			return derp.Wrap(err, "service.template.calculateAllInheritance", "Error calculating inheritance", template.TemplateID)
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
			return model.Template{}, derp.InternalError(
				location,
				"Parent template is not defined",
				"templateId: "+template.TemplateID,
				"parentId: "+parentID,
			)
		}

		parent, err := service.calculateInheritance(parent)

		if err != nil {
			return model.Template{}, derp.Wrap(err, location, "Error calculating inheritance", template.TemplateID, parentID)
		}

		template.Inherit(&parent)
	}

	service.templatePrep[template.TemplateID] = template

	return template, nil
}

// calculateAllowLists calculates the access lists for every Template in the prep area
func (service *Template) calculateAllowLists() error {

	const location = "service.template.calculateAllowLists"

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

// List returns all templates that match the provided criteria
func (service *Template) List(filter func(*model.Template) bool) []form.LookupCode {

	result := []form.LookupCode{}

	if filter == nil {
		filter = func(_ *model.Template) bool { return true }
	}

	for _, template := range service.templates {
		if filter(&template) {
			result = append(result, form.LookupCode{
				Value:       template.TemplateID,
				Label:       template.Label,
				Description: template.Description,
				Icon:        template.Icon,
				Group:       template.Category,
			})
		}
	}

	// Sort templates by Group, then Label
	sort.Slice(result, func(a int, b int) bool {
		if result[a].Group == result[b].Group {
			return result[a].Label < result[b].Label
		}
		return result[a].Group < result[b].Group
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

	return model.NewTemplate(templateID, nil), derp.NotFoundError("sevice.Template.Load", "Template not found", templateID)
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
func (service *Template) ListByContainerLimited(containedByRole string, limitRoles sliceof.String) []form.LookupCode {

	if limitRoles.IsEmpty() {
		return service.ListByContainer(containedByRole)
	}

	filter := func(t *model.Template) bool {
		return t.ContainedBy.Contains(containedByRole) && limitRoles.Contains(t.TemplateRole)
	}

	return service.List(filter)
}

/******************************************
 * Admin Templates
 ******************************************/

func (service *Template) LoadAdmin(templateID string) (model.Template, error) {

	templateID = "admin-" + templateID

	// Try to load the template
	template, err := service.Load(templateID)

	if err != nil {
		return template, derp.Wrap(err, "service.Template.LoadAdmin", "Unable to load admin template", templateID)
	}

	// RULE: Validate Template ContainedBy
	if template.TemplateRole != "admin" {
		return template, derp.InternalError("service.Template.LoadAdmin", "Template must have 'admin' role.", template.TemplateID, template.TemplateRole)
	}

	if !template.ContainedBy.Equal([]string{"admin"}) {
		return template, derp.InternalError("service.Template.LoadAdmin", "Template must be contained by 'admin'", template.TemplateID, template.ContainedBy)
	}

	// Success!
	return template, nil
}
