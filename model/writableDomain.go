package model

import (
	"strings"

	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WritableDomain is a Domain that can be saved: the record plus the journal that data.Object
// requires.  Both halves are stored inline, so the document keeps its shape (see AGENTS.md).
type WritableDomain struct {
	Domain          `bson:",inline"`
	journal.Journal `json:"-" bson:",inline"`
}

// NewWritableDomain returns a fully initialized WritableDomain object
func NewWritableDomain() WritableDomain {
	return WritableDomain{
		Domain: NewDomain(),
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (writableDomain *WritableDomain) GetPointer(name string) (any, bool) {

	switch name {

	case "registrationId":
		return &writableDomain.RegistrationID, true

	case "inboxId":
		return &writableDomain.InboxID, true

	case "outboxId":
		return &writableDomain.OutboxID, true

	case "registrationData":
		return &writableDomain.RegistrationData, true

	case "themeId":
		return &writableDomain.ThemeID, true

	case "label":
		return &writableDomain.Label, true

	case "description":
		return &writableDomain.Description, true

	case "forward":
		return &writableDomain.Forward, true

	case "colorMode":
		return &writableDomain.ColorMode, true

	case "mlsMode":
		return &writableDomain.MLSMode, true

	case "data":
		return &writableDomain.Data, true

	case "themeData":
		return &writableDomain.ThemeData, true

	case "syndication":
		return &writableDomain.Syndication, true

	case "defaultAnonymous":
		return &writableDomain.DefaultAnonymous, true

	case "defaultAuthenticated":
		return &writableDomain.DefaultAuthenticated, true

	case "defaultOwner":
		return &writableDomain.DefaultOwner, true

	case "startupTasks":
		return &writableDomain.StartupTasks, true

	case "stateId":
		return &writableDomain.StateID, true
	}

	return nil, false
}

/*********************************
 * Setter Interfaces
 *********************************/

// SetString writes the named property. Implements schema.StringSetter.
func (writableDomain *WritableDomain) SetString(name string, value string) bool {

	switch name {

	case "domainId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			writableDomain.DomainID = objectID
			return true
		}

	case "iconId":
		if value == "" {
			writableDomain.IconID = primitive.NilObjectID
			return true
		}

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			writableDomain.IconID = objectID
			return true
		}

	case "imageId":
		if value == "" {
			writableDomain.ImageID = primitive.NilObjectID
			return true
		}

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			writableDomain.ImageID = objectID
			return true
		}

	case "mlsGroupIds":
		writableDomain.MLSGroupIDs = strings.Split(value, ",")
		return true

	case "iconUrl":
		return true // Virtual fields can't be set, but don't return an error if someone tries

	case "imageUrl":
		return true // Virtual fields can't be set, but don't return an error if someone tries
	}

	return false
}
