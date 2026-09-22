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
func (domain *WritableDomain) GetPointer(name string) (any, bool) {

	switch name {

	case "registrationId":
		return &domain.RegistrationID, true

	case "inboxId":
		return &domain.InboxID, true

	case "outboxId":
		return &domain.OutboxID, true

	case "registrationData":
		return &domain.RegistrationData, true

	case "themeId":
		return &domain.ThemeID, true

	case "label":
		return &domain.Label, true

	case "description":
		return &domain.Description, true

	case "forward":
		return &domain.Forward, true

	case "colorMode":
		return &domain.ColorMode, true

	case "mlsMode":
		return &domain.MLSMode, true

	case "data":
		return &domain.Data, true

	case "themeData":
		return &domain.ThemeData, true

	case "syndication":
		return &domain.Syndication, true

	case "defaultAnonymous":
		return &domain.DefaultAnonymous, true

	case "defaultAuthenticated":
		return &domain.DefaultAuthenticated, true

	case "defaultOwner":
		return &domain.DefaultOwner, true

	case "startupTasks":
		return &domain.StartupTasks, true

	case "stateId":
		return &domain.StateID, true
	}

	return nil, false
}

/*********************************
 * Setter Interfaces
 *********************************/

// SetString writes the named property. Implements schema.StringSetter.
func (domain *WritableDomain) SetString(name string, value string) bool {

	switch name {

	case "domainId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			domain.DomainID = objectID
			return true
		}

	case "iconId":
		if value == "" {
			domain.IconID = primitive.NilObjectID
			return true
		}

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			domain.IconID = objectID
			return true
		}

	case "imageId":
		if value == "" {
			domain.ImageID = primitive.NilObjectID
			return true
		}

		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			domain.ImageID = objectID
			return true
		}

	case "mlsGroupIds":
		domain.MLSGroupIDs = strings.Split(value, ",")
		return true

	case "iconUrl":
		return true // Virtual fields can't be set, but don't return an error if someone tries

	case "imageUrl":
		return true // Virtual fields can't be set, but don't return an error if someone tries
	}

	return false
}
