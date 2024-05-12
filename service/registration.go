package service

import (
	"html/template"
	"io/fs"
	"sort"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/hjson/hjson-go/v4"
	"github.com/rs/zerolog/log"
)

// Registration service manages new user registrations
type Registration struct {
	templates map[string]model.Registration
	funcMap   template.FuncMap
	mutex     sync.RWMutex
}

// NewRegistration returns a fully initialized Registration service
func NewRegistration(funcMap template.FuncMap) Registration {
	return Registration{
		templates: make(map[string]model.Registration),
		funcMap:   funcMap,
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Add loads a registration definition from a filesystem, and adds it to the in-memory library.
func (service *Registration) Add(registrationID string, filesystem fs.FS, definition []byte) error {

	const location = "service.registration.Add"

	log.Trace().Msg("Registration Service: adding registration: " + registrationID)

	registration := model.NewRegistration(registrationID, service.funcMap)

	// Unmarshal the file into the schema.
	if err := hjson.Unmarshal(definition, &registration); err != nil {
		return derp.Wrap(err, location, "Error loading Schema", registrationID)
	}

	// Load all HTML templates from the filesystem
	if err := loadHTMLTemplateFromFilesystem(filesystem, registration.HTMLTemplate, service.funcMap); err != nil {
		return derp.Wrap(err, location, "Error loading Registration", registrationID)
	}

	// Load all Bundles from the filesystem
	if err := populateBundles(registration.Bundles, filesystem); err != nil {
		return derp.Wrap(err, location, "Error loading Bundles", registrationID)
	}

	// Keep a pointer to the filesystem resources (if present)
	if resources, err := fs.Sub(filesystem, "resources"); err == nil {
		registration.Resources = resources
	}

	// Add the registration into the service library
	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.templates[registration.RegistrationID] = registration

	return nil
}

// List returns all registrations that match the provided criteria
func (service *Registration) List() []form.LookupCode {

	result := []form.LookupCode{}

	for _, registration := range service.templates {
		result = append(result, form.LookupCode{
			Value:       registration.RegistrationID,
			Label:       registration.Label,
			Description: registration.Description,
			Icon:        registration.Icon,
		})
	}

	// Sort registrations by Group, then Label
	sort.Slice(result, func(a int, b int) bool {
		return result[a].Group < result[b].Group
	})

	return result
}

func (service *Registration) Load(registrationID string) (model.Registration, error) {

	// Allow "empty" registration
	if registrationID == "" {
		return model.NewRegistration("", nil), nil
	}

	// READ Mutex to make multi-threaded access safe.
	service.mutex.RLock()
	defer service.mutex.RUnlock()

	// Look in the local cache first
	if registration, ok := service.templates[registrationID]; ok {
		return registration, nil
	}

	return model.NewRegistration(registrationID, nil), derp.NewNotFoundError("sevice.Registration.Load", "Registration not found", registrationID)
}
