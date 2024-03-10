package service

import (
	"html/template"
	"io/fs"
	"sort"
	"strconv"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/hjson/hjson-go/v4"
	"github.com/rs/zerolog/log"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates         set.Map[model.Template]      // map of all templates available within this domain
	templatePrep      set.Map[model.Template]      // temporary map of templates that are being prepared
	locations         sliceof.Object[mapof.String] // Configuration for template directory
	filesystemService Filesystem                   // Filesystem service
	themeService      *Theme                       // Theme Service
	widgetService     *Widget                      // Widget Service
	funcMap           template.FuncMap             // Map of functions to use in golang templates
	mutex             sync.RWMutex                 // Mutext that locks access to the templates structure
	changed           chan bool                    // Channel that is used to signal that a template has changed
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(filesystemService Filesystem, themeService *Theme, widgetService *Widget, funcMap template.FuncMap, locations []mapof.String) *Template {

	service := Template{
		templates:         make(set.Map[model.Template]),
		templatePrep:      make(set.Map[model.Template]),
		locations:         make(sliceof.Object[mapof.String], 0),
		filesystemService: filesystemService,
		themeService:      themeService,
		widgetService:     widgetService,
		funcMap:           funcMap,
		changed:           make(chan bool),
	}

	service.Refresh(locations)

	return &service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *Template) Refresh(locations sliceof.Object[mapof.String]) {

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

	// Start new watchers.
	for _, folder := range service.locations {

		if err := service.filesystemService.Watch(folder, service.changed); err != nil {
			derp.Report(derp.Wrap(err, "service.template.Watch", "Error watching filesystem", folder))
		}
	}

	// All Watchers Started.  Now Listen for Changes
	for range service.changed {
		if err := service.loadTemplates(); err != nil {
			derp.Report(derp.Wrap(err, "service.template.Watch", "Error loading templates from filesystem"))
		}
	}
}

// loadTemplates retrieves the template from the filesystem and parses it into
func (service *Template) loadTemplates() error {

	service.templatePrep = make(set.Map[model.Template])

	// For each configured location...
	for _, location := range service.locations {

		// Get a valid filesystem adapter
		filesystem, err := service.filesystemService.GetFS(location)

		if err != nil {
			derp.Report(derp.Wrap(err, "service.Template.loadTemplates", "Error getting filesystem adapter", location))
			continue
		}

		directories, err := fs.ReadDir(filesystem, ".")

		if err != nil {
			derp.Report(derp.Wrap(err, "service.Template.loadTemplates", "Error reading directory", location))
			continue
		}

		for _, directory := range directories {

			if !directory.IsDir() {
				continue
			}

			directoryName := directory.Name()
			subdirectory, err := fs.Sub(filesystem, directoryName)

			if err != nil {
				derp.Report(derp.Wrap(err, "service.Template.loadTemplates", "Error getting filesystem adapter for sub-directory", location))
				continue
			}

			definitionType, file, err := findDefinition(subdirectory)

			if err != nil {
				derp.Report(derp.Wrap(err, "service.Template.loadTemplates", "Invalid definition", location))
				continue
			}

			switch definitionType {

			// TODO: LOW: Add DefinitionEmail to this.  Will need a *.json file in the email directory.

			case DefinitionTheme:
				if err := service.themeService.Add(directoryName, subdirectory, file); err != nil {
					derp.Report(derp.Wrap(err, "service.Template.loadTemplates", "Error adding theme"))
				}

			case DefinitionWidget:
				if err := service.widgetService.Add(directoryName, subdirectory, file); err != nil {
					derp.Report(derp.Wrap(err, "service.Template.loadTemplates", "Error adding widget"))
				}

			case DefinitionTemplate:
				if err := service.Add(directoryName, subdirectory, file); err != nil {
					derp.Report(derp.Wrap(err, "service.Template.loadTemplates", "Error adding template"))
				}

			default:
				derp.Report(derp.NewInternalError("service.Template.loadTemplates", "Invalid definition", location))
			}
		}
	}

	// Handle inheritance for each template
	for _, template := range service.templatePrep {
		if len(template.Extends) > 0 {
			service.calculateInheritance(template)
		}
	}

	service.themeService.calculateAllInheritance()

	// Assign the prep area to live
	service.mutex.Lock()
	defer service.mutex.Unlock()
	for templateID, template := range service.templatePrep {
		service.templates[templateID] = template
	}

	// Clear out the existing prep area
	service.templatePrep = make(set.Map[model.Template])
	log.Debug().Msg("Template Service: Added/Updated " + strconv.Itoa(len(service.templates)) + " templates")

	return nil
}

func (service *Template) Add(templateID string, filesystem fs.FS, definition []byte) error {

	const location = "service.template.Add"

	log.Trace().Msg("Template Service: adding " + templateID)

	template := model.NewTemplate(templateID, service.funcMap)

	// Unmarshal the file into the schema.
	if err := hjson.Unmarshal(definition, &template); err != nil {
		return derp.Wrap(err, location, "Error loading Schema", templateID)
	}

	// All template schemas also inherit from the main stream schema
	template.Schema.Inherit(schema.New(model.StreamSchema()))

	// Load all HTML templates from the filesystem
	if err := loadHTMLTemplateFromFilesystem(filesystem, template.HTMLTemplate, service.funcMap); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Error loading Template", templateID))
	}

	// Load all Bundles from the filesystem
	if err := populateBundles(template.Bundles, filesystem); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Error loading Bundles", templateID))
	}

	// Keep a pointer to the filesystem resources (if present)
	if resources, err := fs.Sub(filesystem, "resources"); err == nil {
		template.Resources = resources
	}

	// Add the template into the prep library
	service.templatePrep[template.TemplateID] = template

	return nil
}

func (service *Template) calculateInheritance(template model.Template) model.Template {

	if len(template.Extends) == 0 {
		return template
	}

	for _, parentID := range template.Extends {
		if parent, ok := service.templatePrep[parentID]; ok {
			parent = service.calculateInheritance(parent)
			template.Inherit(&parent)
		}
	}

	service.templatePrep[template.TemplateID] = template
	return template
}

/******************************************
 * Common Data Methods
 ******************************************/

// List returns all templates that match the provided criteria
func (service *Template) List(filter func(*model.Template) bool) []form.LookupCode {

	result := []form.LookupCode{}

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

	return model.NewTemplate(templateID, nil), derp.NewNotFoundError("sevice.Template.Load", "Template not found", templateID)
}

/******************************************
 * Custom Queries
 ******************************************/

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
		return template, derp.NewInternalError("service.Template.LoadAdmin", "Template must have 'admin' role.", template.TemplateID, template.TemplateRole)
	}

	if !template.ContainedBy.Equal([]string{"admin"}) {
		return template, derp.NewInternalError("service.Template.LoadAdmin", "Template must be contained by 'admin'", template.TemplateID, template.ContainedBy)
	}

	// Success!
	return template, nil
}

/******************************************
 * Other Data Access Methods
 ******************************************/

// State returns the detailed State information associated with this Stream
func (service *Template) State(templateID string, stateID string) (model.State, error) {

	// Try to find the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
		return model.State{}, derp.Wrap(err, "service.Template.State", "Invalid Template", templateID)
	}

	// Try to find the state data for the state that the stream is in
	state, ok := template.State(stateID)

	if !ok {
		return state, derp.NewInternalError("service.Template.State", "Invalid state", templateID, stateID)
	}

	// Success!
	return state, nil
}

// Schema returns the Schema associated with this Stream
func (service *Template) Schema(templateID string) (schema.Schema, error) {

	// Try to locate the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
		return schema.Schema{}, derp.Wrap(err, "service.Template.Action", "Invalid Template", templateID)
	}

	// Return the Schema defined in this template.
	return template.Schema, nil
}
