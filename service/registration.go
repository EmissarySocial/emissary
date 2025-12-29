package service

import (
	"html/template"
	"io/fs"
	"sort"
	"strings"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/hjson/hjson-go/v4"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	log.Debug().Msg("Registration Service: adding registration: " + registrationID)

	registration := model.NewRegistration(registrationID, service.funcMap)

	// Unmarshal the file into the schema.
	if err := hjson.Unmarshal(definition, &registration); err != nil {
		return derp.Wrap(err, location, "Unable to load Schema", registrationID)
	}

	// Load all HTML templates from the filesystem
	if err := loadHTMLTemplateFromFilesystem(filesystem, registration.HTMLTemplate, service.funcMap); err != nil {
		return derp.Wrap(err, location, "Unable to load Registration", registrationID)
	}

	// Load all Bundles from the filesystem
	if err := populateBundles(registration.Bundles, filesystem); err != nil {
		return derp.Wrap(err, location, "Unable to load Bundles", registrationID)
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

	return model.NewRegistration(registrationID, nil), derp.NotFoundError("sevice.Registration.Load", "Registration not found", registrationID)
}

/******************************************
 * User Registration
 ******************************************/

func (service *Registration) Validate(session data.Session, userService *User, domain *model.Domain, txn model.RegistrationTxn) error {

	// TODO: Once we have a single "factory" architecture, we should remove userService as an argument to this function

	const location = "service.Registration.Validate"

	// Validate basic transaction values (name, email, username, userID, secret)
	if secret := domain.RegistrationData.GetString("secret"); txn.IsInvalid(secret) {
		return derp.BadRequestError(location, "Invalid Registration. Please sign up again.", txn)
	}

	user := model.NewUser()

	// UserID must not already exist in the database
	if userID, err := primitive.ObjectIDFromHex(txn.UserID); err != nil {
		return derp.BadRequestError(location, "Invalid UserID", txn.UserID)

	} else if err := userService.LoadByID(session, userID, &user); !derp.IsNotFound(err) {
		return derp.BadRequestError(location, "UserID already exists. Please sign up again.")
	}

	// Username must not already exist in the database
	if err := userService.LoadByUsername(session, txn.Username, &user); !derp.IsNotFound(err) {
		return derp.BadRequestError(location, "Username taken. Please choose again.")
	}

	// This transaction appears to be valid.
	return nil
}

func (service *Registration) Register(session data.Session, groupService *Group, userService *User, domain *model.Domain, txn model.RegistrationTxn) (model.User, error) {

	const location = "service.Registration.Register"

	// Get the domain and registration information
	registration, err := service.Load(domain.RegistrationID)

	if err != nil {
		return model.User{}, derp.Wrap(err, location, "Unable to load registration")
	}

	if registration.IsZero() {
		return model.User{}, derp.NotFoundError(location, "Registration not found")
	}

	// Validate the transaction
	if err := service.Validate(session, userService, domain, txn); err != nil {
		return model.User{}, derp.Wrap(err, location, "Invalid Registration Transaction")
	}

	// Copy Transaction data into a new User object
	user := model.NewUser()
	if err := service.setUserData(session, groupService, domain, &user, txn, registration.AllowedFields); err != nil {
		return model.User{}, derp.Wrap(err, location, "Unable to set user data")
	}

	// If defined in the registration data, set the User's Inbox Template
	if inboxTemplate := domain.RegistrationData.GetString("inboxTemplate"); inboxTemplate != "" {
		user.InboxTemplate = inboxTemplate
	}

	// If defined in the registration data, set the User's Inbox Template
	if outboxTemplate := domain.RegistrationData.GetString("outboxTemplate"); outboxTemplate != "" {
		user.OutboxTemplate = outboxTemplate
	}

	// Try to save the User to the database
	if err := userService.Save(session, &user, "Created by Online Registration"); err != nil {
		return model.User{}, derp.Wrap(err, location, "Unable to create new User")
	}

	// Word to your mother.
	return user, nil
}

// UpdateRegistration updates an existing User with new data from a Registration Transaction
func (service *Registration) UpdateRegistration(session data.Session, groupService *Group, userService *User, domain *model.Domain, source string, sourceID string, txn model.RegistrationTxn) error {

	const location = "service.Registration.UpdateRegistration"

	// Get the Registration object
	registration, err := service.Load(domain.RegistrationID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load registration")
	}

	if registration.IsZero() {
		return derp.NotFoundError(location, "Registration not found")
	}

	// Validate the Transaction
	if secret := domain.RegistrationData.GetString("secret"); secret == "" {
		return derp.NotFoundError(location, "This registration form requires a secret key")
	} else if txn.IsInvalid(secret) {
		return derp.BadRequestError(location, "Invalid Registration Transaction", txn)
	}

	// Try to locate the user from their remote ID
	user := model.NewUser()
	err = userService.LoadByMapID(session, source, sourceID, &user)

	// If not found, then create a new User
	if derp.IsNotFound(err) {
		if _, inner := service.Register(session, groupService, userService, domain, txn); inner != nil {
			return derp.Wrap(inner, location, "Unable to create new User from registration transaction")
		}
		return nil
	}

	// Squalk all other errors
	if err != nil {
		return derp.Wrap(err, location, "Unable to load user", source, sourceID)
	}

	// Update user data from the transaction
	if err := service.setUserData(session, groupService, domain, &user, txn, registration.AllowedFields); err != nil {
		return derp.Wrap(err, location, "Unable to set user data")
	}

	// Try to save the User to the database
	if err := userService.Save(session, &user, "Created by Online Registration"); err != nil {
		return derp.Wrap(err, location, "Unable to create new User")
	}

	// Word to your mother.
	return nil
}

// setUserData copies all allowed fields from the Transaction into the User, and silently warns if any field names are not recognized
func (service *Registration) setUserData(session data.Session, groupService *Group, domain *model.Domain, user *model.User, txn model.RegistrationTxn, allowedFields []string) error {

	const location = "service.Registration.setUserData"

	for _, fieldName := range allowedFields {

		switch fieldName {

		case "displayName":

			if txn.DisplayName != "" {
				user.DisplayName = txn.DisplayName
			}

		case "emailAddress":

			if txn.EmailAddress != "" {
				user.EmailAddress = txn.EmailAddress
			}

		case "username":

			if txn.Username != "" {
				user.Username = txn.Username
			}

		case "password":

			if user.IsNew() && txn.Password != "" {
				user.Password = txn.Password
			}

		case "stateId":

			if txn.StateID != "" {
				user.StateID = txn.StateID
			}

		case "inboxTemplate":

			if txn.InboxTemplate != "" {
				user.InboxTemplate = txn.InboxTemplate
			}

		case "outboxTemplate":

			if txn.OutboxTemplate != "" {
				user.OutboxTemplate = txn.OutboxTemplate
			}

		case "addGroups":
			if err := service.addGroups(session, groupService, user, txn.AddGroups); err != nil {
				return derp.Wrap(err, location, "Error adding user to group", txn.AddGroups)
			}

		case "removeGroups":
			if err := service.removeGroups(session, groupService, user, txn.RemoveGroups); err != nil {
				return derp.Wrap(err, location, "Error adding user to group", txn.RemoveGroups)
			}

		default:
			derp.Report(derp.InternalError(location, "Unknown field", fieldName))
		}
	}

	// Settings from the Domain override any other settings

	if stateID := domain.RegistrationData.GetString("stateId"); stateID != "" {
		user.StateID = stateID
	}

	if inboxTemplate := domain.RegistrationData.GetString("inboxTemplate"); inboxTemplate != "" {
		user.InboxTemplate = inboxTemplate
	}

	if outboxTemplate := domain.RegistrationData.GetString("outboxTemplate"); outboxTemplate != "" {
		user.OutboxTemplate = outboxTemplate
	}

	if addGroups := domain.RegistrationData.GetString("addGroups"); addGroups != "" {
		if err := service.addGroups(session, groupService, user, addGroups); err != nil {
			return derp.Wrap(err, location, "Error adding user to group", addGroups)
		}
	}

	if removeGroups := domain.RegistrationData.GetString("removeGroups"); removeGroups != "" {
		if err := service.removeGroups(session, groupService, user, removeGroups); err != nil {
			return derp.Wrap(err, location, "Error adding user to group", removeGroups)
		}
	}

	return nil
}

// addGroup adds a User to a Group (using either the group.GroupID or the group.Token)
func (service *Registration) addGroups(session data.Session, groupService *Group, user *model.User, groupIDs string) error {

	const location = "service.Registration.addGroup"

	if groupIDs == "" {
		return nil
	}

	for _, token := range strings.Split(groupIDs, ",") {

		// Locate the Group using ID or Token
		token = strings.TrimSpace(token)
		group := model.NewGroup()
		if err := groupService.LoadByToken(session, token, &group); err != nil {
			return derp.Wrap(err, location, "Unable to load group", token)
		}

		// Add the User to the Group
		user.AddGroup(group.GroupID)

	}
	return nil
}

// removeGroup removes a User from a Group (using either the group.GroupID or the group.Token)
func (service *Registration) removeGroups(session data.Session, groupService *Group, user *model.User, groupIDs string) error {

	const location = "service.Registration.removeGroup"

	if groupIDs == "" {
		return nil
	}

	for _, token := range strings.Split(groupIDs, ",") {

		// Locate the Group using ID or Token
		token = strings.TrimSpace(token)
		group := model.NewGroup()
		if err := groupService.LoadByToken(session, token, &group); err != nil {
			return derp.Wrap(err, location, "Unable to load group", token)
		}

		// Remove the User from the Group
		user.RemoveGroup(group.GroupID)
	}

	return nil
}
