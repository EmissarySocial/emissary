package service

import (
	"github.com/benpate/activitystream/reader"
	"github.com/benpate/activitystream/vocabulary"
	"github.com/benpate/activitystream/writer"
	"github.com/benpate/data"
	"github.com/benpate/ghost/model"
)

// ActivityPub service manages all interactions with ActivityPub objects
type ActivityPub struct {
	factory Factory
	session data.Session
}

// GetInbox returns inbox information for the requested Actor
func (service ActivityPub) GetInbox(actor model.Actor) writer.Collection {
	return writer.Collection{}
}

// PostInbox adds a new item to a User's inbox
func (service ActivityPub) PostInbox(info reader.Object) error {

	switch info.Type() {

	case vocabulary.ActivityTypeAccept:

	case vocabulary.ActivityTypeAdd:

	case vocabulary.ActivityTypeAnnounce:

	case vocabulary.ActivityTypeArrive:

	case vocabulary.ActivityTypeBlock:

	case vocabulary.ActivityTypeCreate:

	case vocabulary.ActivityTypeDelete:

	case vocabulary.ActivityTypeDislike:

	case vocabulary.ActivityTypeFlag:

	case vocabulary.ActivityTypeFollow:

	case vocabulary.ActivityTypeIgnore:

	case vocabulary.ActivityTypeInvite:

	case vocabulary.ActivityTypeJoin:

	case vocabulary.ActivityTypeLeave:

	case vocabulary.ActivityTypeLike:

	case vocabulary.ActivityTypeListen:

	case vocabulary.ActivityTypeMove:

	case vocabulary.ActivityTypeOffer:

	case vocabulary.ActivityTypeQuestion:

	case vocabulary.ActivityTypeRead:

	case vocabulary.ActivityTypeReject:

	case vocabulary.ActivityTypeRemove:

	case vocabulary.ActivityTypeTentativeAccept:

	case vocabulary.ActivityTypeTentativeReject:

	case vocabulary.ActivityTypeTravel:

	case vocabulary.ActivityTypeUndo:

	case vocabulary.ActivityTypeUpdate:

	case vocabulary.ActivityTypeView:
	}

	return nil
}

// GetOutbox returns outbox information for the requested Actor
func (service ActivityPub) GetOutbox(actor model.Actor) writer.Collection {
	return writer.Collection{}
}

// PostOutbox adds a new item to a User's outbox
func (service ActivityPub) PostOutbox(info reader.Object) error {

	switch info.Type() {

	case vocabulary.ActivityTypeAccept:

	case vocabulary.ActivityTypeAdd:

	case vocabulary.ActivityTypeAnnounce:

	case vocabulary.ActivityTypeArrive:

	case vocabulary.ActivityTypeBlock:

	case vocabulary.ActivityTypeCreate:

	case vocabulary.ActivityTypeDelete:

	case vocabulary.ActivityTypeDislike:

	case vocabulary.ActivityTypeFlag:

	case vocabulary.ActivityTypeFollow:

	case vocabulary.ActivityTypeIgnore:

	case vocabulary.ActivityTypeInvite:

	case vocabulary.ActivityTypeJoin:

	case vocabulary.ActivityTypeLeave:

	case vocabulary.ActivityTypeLike:

	case vocabulary.ActivityTypeListen:

	case vocabulary.ActivityTypeMove:

	case vocabulary.ActivityTypeOffer:

	case vocabulary.ActivityTypeQuestion:

	case vocabulary.ActivityTypeRead:

	case vocabulary.ActivityTypeReject:

	case vocabulary.ActivityTypeRemove:

	case vocabulary.ActivityTypeTentativeAccept:

	case vocabulary.ActivityTypeTentativeReject:

	case vocabulary.ActivityTypeTravel:

	case vocabulary.ActivityTypeUndo:

	case vocabulary.ActivityTypeUpdate:

	case vocabulary.ActivityTypeView:

	}

	return nil
}
