package service

import (
	"encoding/json"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// importedObjectFinder finds an object that a User's import has already brought to this server
type importedObjectFinder interface {

	// findImportedObject returns the object's current JSON-LD and its publish date in Unix
	// milliseconds, and FALSE when this User's import holds no object for the remote URL
	findImportedObject(session data.Session, userID primitive.ObjectID, remoteURL string) (mapof.Any, int64, bool)
}

// Import is a part of the "Importable" interface, and saves an exported OutboxItem to a User's outbox
func (service *Outbox2) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Outbox2.Import"

	// Unmarshal the JSON document into a new OutboxItem
	item := model.NewOutboxItem()

	if err := json.Unmarshal(document, &item); err != nil {
		return derp.Wrap(err, location, "Parsing remote document", derp.WithBadRequest())
	}

	// RULE: An account export holds a User's own outbox, and nothing else
	if item.ActorType != model.ActorTypeUser {
		return derp.BadRequest(location, "Only a User's outbox can be imported", "actorType: "+item.ActorType)
	}

	if err := service.saveImported(session, importItem, user, &item); err != nil {
		return derp.Wrap(err, location, "Saving imported activity")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes an imported OutboxItem
func (service *Outbox2) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Outbox2.UndoImport"

	// Load the activity, scoped to the outbox of the User who imported it
	item := model.NewOutboxItem()
	err := service.LoadByID(session, model.ActorTypeUser, importItem.UserID, importItem.LocalID, &item)

	// An activity that is already gone needs no undoing
	if derp.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return derp.Wrap(err, location, "Loading imported activity", "activityId: "+importItem.LocalID.Hex())
	}

	if err := service.Delete(session, &item, "Import undone"); err != nil {
		return derp.Wrap(err, location, "Deleting imported activity", "activityId: "+importItem.LocalID.Hex())
	}

	return nil
}

// LegacyImporter returns the Importable that converts an exported legacy OutboxMessage into an OutboxItem
func (service *Outbox2) LegacyImporter() Importable {
	return outboxMessageImporter{outbox2: service}
}

// outboxMessageImporter imports the `emissary:outboxMessage` collection, which servers that
// still use the legacy outbox export, into Outbox2
type outboxMessageImporter struct {
	outbox2 *Outbox2
}

// Import is a part of the "Importable" interface, and saves an exported OutboxMessage as an OutboxItem
func (importer outboxMessageImporter) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.outboxMessageImporter.Import"

	// Unmarshal the JSON document into a legacy OutboxMessage
	message := model.NewOutboxMessage()

	if err := json.Unmarshal(document, &message); err != nil {
		return derp.Wrap(err, location, "Parsing remote document", derp.WithBadRequest())
	}

	// RULE: An account export holds a User's own outbox, and nothing else
	if message.ActorType != model.ActorTypeUser {
		return derp.BadRequest(location, "Only a User's outbox can be imported", "actorType: "+message.ActorType)
	}

	item := outboxItemFromMessage(&message)

	if err := importer.outbox2.saveImported(session, importItem, user, &item); err != nil {
		return derp.Wrap(err, location, "Saving imported activity")
	}

	// Old wine, new bottle
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes an imported OutboxItem
func (importer outboxMessageImporter) UndoImport(session data.Session, importItem *model.ImportItem) error {
	return importer.outbox2.UndoImport(session, importItem)
}

// outboxItemFromMessage returns the OutboxItem that carries a legacy OutboxMessage's activity.
// The legacy outbox stored no addressing and no object, so the activity names its object by URL.
func outboxItemFromMessage(message *model.OutboxMessage) model.OutboxItem {

	item := model.NewOutboxItem()
	item.ActivityID = message.OutboxMessageID
	item.ActorType = message.ActorType
	item.ActorID = message.ActorID
	item.URL = message.ActivityPubURL()
	item.Permissions = message.Permissions
	item.CreateDate = message.CreateDate

	item.Activity = mapof.Any{
		vocab.AtContext:      vocab.ContextTypeActivityStreams,
		vocab.PropertyType:   message.ActivityType,
		vocab.PropertyActor:  message.ActorURL,
		vocab.PropertyObject: message.ObjectID,
	}

	// The legacy outbox served `to: Public` for any message that everyone could read
	if message.Permissions.IsAnonymous() {
		item.Activity[vocab.PropertyTo] = []string{vocab.NamespacePublic}
	}

	return item
}

// saveImported stores an activity from another server in a User's outbox, under this server's
// URLs. It sends nothing: an import must never deliver the User's history a second time.
func (service *Outbox2) saveImported(session data.Session, importItem *model.ImportItem, user *model.User, item *model.OutboxItem) error {

	const location = "service.Outbox2.saveImported"

	// Record where the activity came from, and give it a new identity here
	importItem.RemoteID = item.ActivityID
	importItem.RemoteURL = item.URL
	importItem.LocalID = primitive.NewObjectID()

	remoteActorURL := item.Activity.GetString(vocab.PropertyActor)
	localActorURL := service.locator.UserURL(user.UserID)

	// RULE: The importing User owns the activity, whatever actor the document names
	item.ActivityID = importItem.LocalID
	item.ActorType = model.ActorTypeUser
	item.ActorID = user.UserID
	item.URL = service.locator.ActivityURL(model.ActorTypeUser, user.UserID, item.ActivityID)
	item.Activity[vocab.PropertyID] = item.URL
	item.Activity[vocab.PropertyActor] = localActorURL

	// Point the addressing at this server's copy of the User, such as their followers collection
	replaceActorInAddressing(item.Activity, remoteActorURL, localActorURL)

	// Replace an object that this import already brought over with its copy on this server
	createDate := service.replaceImportedObject(session, user.UserID, item)

	// RULE: Describe the activity from its own content, never from the document's claims about it
	item.ActivityType = item.Activity.GetString(vocab.PropertyType)
	item.ObjectID = rawObjectID(item.Activity)
	item.CalcRecipients()

	// Keep the original date, so that the outbox stays in its original order. A new journal
	// drops everything else, such as a delete date. A zero date is stamped by Save.
	item.Journal = journal.Journal{CreateDate: createDate}

	if createDate > 0 {
		item.Activity[vocab.PropertyPublished] = datetime.FromUnixMilli(createDate)
	}

	if err := service.Save(session, item, "Imported"); err != nil {
		return derp.Wrap(err, location, "Saving imported OutboxItem", "activityId: "+item.ActivityID.Hex())
	}

	importItem.LocalURL = item.URL

	return nil
}

// replaceImportedObject swaps the activity's object for this server's copy when the User's import
// already brought it over, and returns the date to file the activity under
func (service *Outbox2) replaceImportedObject(session data.Session, userID primitive.ObjectID, item *model.OutboxItem) int64 {

	remoteObjectURL := rawObjectID(item.Activity)

	// An activity with no object keeps whatever date it was exported with
	if remoteObjectURL == "" {
		return item.CreateDate
	}

	object, publishDate, found := service.importedObjects.findImportedObject(session, userID, remoteObjectURL)

	// Any other object stays as exported, and still names the server it came from
	if !found {
		return item.CreateDate
	}

	item.Activity[vocab.PropertyObject] = object

	// A legacy export carries no date, so fall back to the date its object was published
	if item.CreateDate > 0 {
		return item.CreateDate
	}

	return publishDate
}

// rawObjectID returns the id of an activity's object without loading anything
func rawObjectID(activity mapof.Any) string {

	switch object := activity[vocab.PropertyObject].(type) {

	case string:
		return object

	case map[string]any:
		return mapof.Any(object).GetString(vocab.PropertyID)

	case mapof.Any:
		return object.GetString(vocab.PropertyID)

	default:
		return ""
	}
}

// replaceActorInAddressing rewrites every recipient that is the remote actor, or a collection
// beneath it, to the same path beneath the local actor
func replaceActorInAddressing(activity mapof.Any, remoteActorURL string, localActorURL string) {

	// RULE: An empty actor would match every recipient
	if remoteActorURL == "" {
		return
	}

	for _, name := range addressingProperties {

		recipients := activity.GetSliceOfString(name)

		if len(recipients) == 0 {
			continue
		}

		for index, recipient := range recipients {

			if recipient == remoteActorURL {
				recipients[index] = localActorURL
				continue
			}

			if path, found := strings.CutPrefix(recipient, remoteActorURL+"/"); found {
				recipients[index] = localActorURL + "/" + path
			}
		}

		activity[name] = recipients
	}
}

// importedStreamFinder finds the Streams that a User's import has brought to this server
type importedStreamFinder struct {
	importItemService *ImportItem
	streamService     *Stream
}

// findImportedObject returns the current JSON-LD and publish date of the Stream that this User
// imported from the remote URL
func (finder importedStreamFinder) findImportedObject(session data.Session, userID primitive.ObjectID, remoteURL string) (mapof.Any, int64, bool) {

	// Find the record of the import, by the URL the object had on the other server
	importItem := model.NewImportItem()

	if err := finder.importItemService.LoadByRemoteURL(session, remoteURL, &importItem); err != nil {
		return nil, 0, false
	}

	// RULE: Only this User's own import counts. Another User's import of the same URL is not theirs to embed.
	if importItem.UserID != userID {
		return nil, 0, false
	}

	// RULE: Only a Stream that has finished importing has a copy here
	if importItem.Type != "emissary:stream" {
		return nil, 0, false
	}

	if importItem.LocalURL == "" {
		return nil, 0, false
	}

	stream := model.NewStream()

	if err := finder.streamService.LoadByID(session, importItem.LocalID, &stream); err != nil {
		return nil, 0, false
	}

	// Stream dates are in seconds; the journal counts milliseconds
	return finder.streamService.JSONLD(session, &stream), publishDateMillis(&stream), true
}

// publishDateMillis returns a Stream's publish date in Unix milliseconds, or zero when it is not published
func publishDateMillis(stream *model.Stream) int64 {

	if !stream.IsPublished() {
		return 0
	}

	return stream.PublishDate * 1000
}
