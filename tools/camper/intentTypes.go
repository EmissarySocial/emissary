package camper

// FEP-3b86 Activity Intent relation types, one per ActivityStreams activity.
// Servers advertise these as WebFinger link relations to declare which
// intents they can complete on a user's behalf.
const (
	IntentTypeAccept          = "https://w3id.org/fep/3b86/Accept"
	IntentTypeAdd             = "https://w3id.org/fep/3b86/Add"
	IntentTypeAnnounce        = "https://w3id.org/fep/3b86/Announce"
	IntentTypeArrive          = "https://w3id.org/fep/3b86/Arrive"
	IntentTypeBlock           = "https://w3id.org/fep/3b86/Block"
	IntentTypeCreate          = "https://w3id.org/fep/3b86/Create"
	IntentTypeDelete          = "https://w3id.org/fep/3b86/Delete"
	IntentTypeDislike         = "https://w3id.org/fep/3b86/Dislike"
	IntentTypeFlag            = "https://w3id.org/fep/3b86/Flag"
	IntentTypeFollow          = "https://w3id.org/fep/3b86/Follow"
	IntentTypeIgnore          = "https://w3id.org/fep/3b86/Ignore"
	IntentTypeInvite          = "https://w3id.org/fep/3b86/Invite"
	IntentTypeJoin            = "https://w3id.org/fep/3b86/Join"
	IntentTypeLeave           = "https://w3id.org/fep/3b86/Leave"
	IntentTypeLike            = "https://w3id.org/fep/3b86/Like"
	IntentTypeListen          = "https://w3id.org/fep/3b86/Listen"
	IntentTypeMove            = "https://w3id.org/fep/3b86/Move"
	IntentTypeOffer           = "https://w3id.org/fep/3b86/Offer"
	IntentTypeQuestion        = "https://w3id.org/fep/3b86/Question"
	IntentTypeRead            = "https://w3id.org/fep/3b86/Read"
	IntentTypeReject          = "https://w3id.org/fep/3b86/Reject"
	IntentTypeRemove          = "https://w3id.org/fep/3b86/Remove"
	IntentTypeTentativeAccept = "https://w3id.org/fep/3b86/TentativeAccept"
	IntentTypeTentativeReject = "https://w3id.org/fep/3b86/TentativeReject"
	IntentTypeTravel          = "https://w3id.org/fep/3b86/Travel"
	IntentTypeUndo            = "https://w3id.org/fep/3b86/Undo"
	IntentTypeUpdate          = "https://w3id.org/fep/3b86/Update"
	IntentTypeView            = "https://w3id.org/fep/3b86/View"
)

// IntentTypeObject is a special case for opening an Object without sending a "View" activity
const IntentTypeObject = "https://w3id.org/fep/3b86/Object"
