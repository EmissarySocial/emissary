package upgrades

import (
	"context"
	"errors"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Version39 copies every row of the legacy Outbox collection into OutboxItem, with the same _id,
// createDate, objectId and permissions, and leaves alone any row that OutboxItem already holds
func Version39(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version39"

	// The legacy rows stay where they are, because an older server still reads them during a
	// rolling deploy. A row that cannot be copied is reported and left behind; only a database
	// failure stops the slot, which then runs again on the next start.

	fmt.Println("... Version 39")

	// Oldest first, so that when two rows share a URL the older one keeps it
	findOptions := options.Find().SetSort(bson.D{{Key: "_id", Value: 1}})
	cursor, err := session.Collection("Outbox").Find(ctx, bson.M{}, findOptions)

	if err != nil {
		return derp.Wrap(err, location, "Reading legacy Outbox rows")
	}

	// A failed Close leaves the server to time the cursor out, which costs nothing here
	defer func() { _ = cursor.Close(context.WithoutCancel(ctx)) }()

	// Copy each row on its own, so that one bad row does not hold back the others
	copier := newLegacyOutboxCopier(session)

	for cursor.Next(ctx) {
		if err := copier.copy(ctx, cursor.Current); err != nil {
			return derp.Wrap(err, location, "Copying legacy Outbox row")
		}
	}

	if err := cursor.Err(); err != nil {
		return derp.Wrap(err, location, "Reading legacy Outbox rows")
	}

	log.Info().Str("database", session.Name()).Int("copied", copier.copied).Int("present", copier.present).Int("skipped", copier.skipped).Msg("Copied the legacy Outbox into OutboxItem")

	// Two outboxes enter, one outbox leaves
	return nil
}

// legacyOutboxCopier copies legacy Outbox rows into OutboxItem, and remembers each actor's URL and
// each URL it has written
type legacyOutboxCopier struct {
	database  *mongo.Database
	target    *mongo.Collection
	actorURLs map[string]string
	urls      map[string]primitive.ObjectID
	copied    int
	present   int
	skipped   int
}

// newLegacyOutboxCopier returns a copier that writes into the database's OutboxItem collection
func newLegacyOutboxCopier(database *mongo.Database) *legacyOutboxCopier {
	return &legacyOutboxCopier{
		database:  database,
		target:    database.Collection("OutboxItem"),
		actorURLs: make(map[string]string),
		urls:      make(map[string]primitive.ObjectID),
	}
}

// copy copies one legacy row. It returns an error only when the database fails; a row that cannot
// be copied is reported and counted as skipped.
func (copier *legacyOutboxCopier) copy(ctx context.Context, raw bson.Raw) error {

	const location = "queries.upgrades.legacyOutboxCopier.copy"

	// Read the row in any shape that Emissary has written
	row := legacyOutboxRow{}

	if err := bson.Unmarshal(raw, &row); err != nil {
		copier.skip(derp.Wrap(err, location, "Decoding row", "_id: "+raw.Lookup("_id").String()))
		return nil
	}

	// RULE: A deleted row is in no outbox, so it is not copied
	if row.DeleteDate > 0 {
		copier.skipped++
		return nil
	}

	message, err := row.message()

	if err != nil {
		copier.skip(derp.Wrap(err, location, "Reading row", "activityId: "+row.ID.Hex()))
		return nil
	}

	// RULE: Only User and Stream actors keep an outbox, so a Search's rows stay behind
	switch message.ActorType {
	case model.ActorTypeUser, model.ActorTypeStream:
	default:
		copier.skipped++
		return nil
	}

	// RULE: A row that names no actor cannot be served by anyone
	if message.ActorID.IsZero() {
		copier.skip(derp.BadRequest(location, "Row names no actor", "activityId: "+row.ID.Hex()))
		return nil
	}

	// Name the actor by URL, which BUG-146 left empty on some rows
	actorURL, err := copier.actorURL(ctx, message)

	if err != nil {
		return derp.Wrap(err, location, "Finding the actor's URL", "activityId: "+row.ID.Hex())
	}

	if actorURL == "" {
		copier.skip(derp.NotFound(location, "Row's actor no longer exists", "activityId: "+row.ID.Hex(), "actorType: "+message.ActorType, "actorId: "+message.ActorID.Hex()))
		return nil
	}

	message.ActorURL = actorURL
	item := outboxItemFromLegacy(message)

	// RULE: Two legacy rows can carry one activityUrl, and idx_OutboxItem_URL admits only one. Check
	// here, because on a database upgrading from v0.7.0 the index is not built until after this slot.
	if holder, exists := copier.urls[item.URL]; exists {
		copier.skip(derp.Conflict(location, "An older legacy row holds this row's URL", "activityId: "+row.ID.Hex(), "heldBy: "+holder.Hex(), "url: "+item.URL))
		return nil
	}

	copier.urls[item.URL] = item.ActivityID

	return copier.write(ctx, item)
}

// skip reports why a row was not copied, and counts it
func (copier *legacyOutboxCopier) skip(err error) {
	derp.Report(err)
	copier.skipped++
}

// legacyOutboxRow is one row of the legacy Outbox collection, in any shape that Emissary has written.
// ParentID, ParentType and URL are the names that v0.7.0 used (BUG-304).
type legacyOutboxRow struct {
	ID           primitive.ObjectID `bson:"_id"`
	ActorID      primitive.ObjectID `bson:"actorId"`
	ActorType    string             `bson:"actorType"`
	ActorURL     string             `bson:"actorUrl"`
	ActivityType string             `bson:"activityType"`
	ActivityURL  string             `bson:"activityUrl"`
	ObjectID     string             `bson:"objectId"`
	Permissions  bson.RawValue      `bson:"permissions"`
	ParentID     primitive.ObjectID `bson:"parentId"`
	ParentType   string             `bson:"parentType"`
	URL          string             `bson:"url"`
	CreateDate   int64              `bson:"createDate"`
	UpdateDate   int64              `bson:"updateDate"`
	DeleteDate   int64              `bson:"deleteDate"`
}

// message returns the row as an OutboxMessage, reading the v0.7.0 names wherever the current ones
// are empty. It returns an error when the row's permissions are not a list of IDs.
func (row legacyOutboxRow) message() (model.OutboxMessage, error) {

	const location = "queries.upgrades.legacyOutboxRow.message"

	message := model.OutboxMessage{
		OutboxMessageID: row.ID,
		ActorID:         row.ActorID,
		ActorType:       row.ActorType,
		ActorURL:        row.ActorURL,
		ActivityType:    row.ActivityType,
		ActivityURL:     row.ActivityURL,
		ObjectID:        row.ObjectID,
		Permissions:     model.NewPermissions(),
		Journal: journal.Journal{
			CreateDate: row.CreateDate,
			UpdateDate: row.UpdateDate,
		},
	}

	// v0.7.0 named the actor as the row's parent, and the object by its url (BUG-304)
	if message.ActorID.IsZero() {
		message.ActorID = row.ParentID
	}

	if message.ActorType == "" {
		message.ActorType = row.ParentType
	}

	if message.ObjectID == "" {
		message.ObjectID = row.URL
	}

	// Every row written before activityType existed is a Create (BUG-304)
	if message.ActivityType == "" {
		message.ActivityType = vocab.ActivityTypeCreate
	}

	// RULE: A row written before permissions existed was served to everyone, which slot 19 once
	// recorded as anonymous (BUG-304). A null list was served to no one, so it stays empty.
	switch row.Permissions.Type {

	case 0:
		message.Permissions = model.NewAnonymousPermissions()

	case bson.TypeArray:
		if err := row.Permissions.Unmarshal(&message.Permissions); err != nil {
			return model.OutboxMessage{}, derp.Wrap(err, location, "Reading permissions")
		}
	}

	return message, nil
}

// actorURL returns the URL of a row's actor: the one the row carries, or else the User's profile URL
// or the Stream's URL. It is empty when the actor no longer exists.
func (copier *legacyOutboxCopier) actorURL(ctx context.Context, message model.OutboxMessage) (string, error) {

	const location = "queries.upgrades.legacyOutboxCopier.actorURL"

	if message.ActorURL != "" {
		return message.ActorURL, nil
	}

	// Each actor is looked up once
	key := message.ActorType + ":" + message.ActorID.Hex()

	if actorURL, exists := copier.actorURLs[key]; exists {
		return actorURL, nil
	}

	collectionName, field := "User", "profileUrl"

	if message.ActorType == model.ActorTypeStream {
		collectionName, field = "Stream", "url"
	}

	record := bson.M{}
	findOptions := options.FindOne().SetProjection(bson.M{field: 1})

	if err := copier.database.Collection(collectionName).FindOne(ctx, bson.M{"_id": message.ActorID}, findOptions).Decode(&record); err != nil {

		// A missing actor has no URL, and its rows stay behind
		if errors.Is(err, mongo.ErrNoDocuments) {
			copier.actorURLs[key] = ""
			return "", nil
		}

		return "", derp.Wrap(err, location, "Loading actor", "actorType: "+message.ActorType, "actorId: "+message.ActorID.Hex())
	}

	// A missing or non-string value names no URL
	actorURL, _ := record[field].(string)
	copier.actorURLs[key] = actorURL

	return actorURL, nil
}

// outboxItemFromLegacy returns the OutboxItem that serves a legacy message exactly as the legacy
// outbox served it, with its object named by URL
func outboxItemFromLegacy(message model.OutboxMessage) model.OutboxItem {

	item := model.OutboxItem{
		ActivityID:   message.OutboxMessageID,
		ActorID:      message.ActorID,
		ActorType:    message.ActorType,
		URL:          message.ActivityPubURL(),
		ActivityType: message.ActivityType,
		ObjectID:     message.ObjectID,
		Activity:     message.GetJSONLD(),
		Permissions:  message.Permissions,
		Journal:      message.Journal,
	}

	item.CalcRecipients()

	return item
}

// write stores an item under its _id unless OutboxItem already holds that _id, and reports an item
// whose URL another row already holds
func (copier *legacyOutboxCopier) write(ctx context.Context, item model.OutboxItem) error {

	const location = "queries.upgrades.legacyOutboxCopier.write"

	// RULE: A row that cannot be encoded is skipped, never returned, or the slot would fail on every start
	fields, err := insertFields(item)

	if err != nil {
		copier.skip(derp.Wrap(err, location, "Encoding row", "activityId: "+item.ActivityID.Hex()))
		return nil
	}

	// RULE: Never overwrite. A row that is already in OutboxItem was written by the current code.
	filter := bson.M{"_id": item.ActivityID}
	update := bson.M{"$setOnInsert": fields}

	result, err := copier.target.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))

	if err != nil {

		// Another server copied this row a moment ago, or another row already holds its URL
		if mongo.IsDuplicateKeyError(err) {
			return copier.duplicate(ctx, item)
		}

		return derp.Wrap(err, location, "Copying row", "activityId: "+item.ActivityID.Hex())
	}

	if result.UpsertedCount == 0 {
		copier.present++
		return nil
	}

	copier.copied++
	return nil
}

// insertFields returns an item's fields as they are stored, without its _id, which the upsert's
// filter supplies
func insertFields(item model.OutboxItem) (bson.D, error) {

	const location = "queries.upgrades.insertFields"

	// Neither error can happen for an item from outboxItemFromLegacy, whose activity holds only
	// strings and string lists; both are checked because the driver reports them
	encoded, err := bson.Marshal(item)

	if err != nil {
		return nil, derp.Wrap(err, location, "Encoding OutboxItem")
	}

	document := bson.D{}

	if err := bson.Unmarshal(encoded, &document); err != nil {
		return nil, derp.Wrap(err, location, "Decoding OutboxItem")
	}

	fields := make(bson.D, 0, len(document))

	for _, element := range document {
		if element.Key != "_id" {
			fields = append(fields, element)
		}
	}

	return fields, nil
}

// duplicate settles a duplicate key: the row is present when its _id is, and otherwise another row
// holds its URL, which is reported
func (copier *legacyOutboxCopier) duplicate(ctx context.Context, item model.OutboxItem) error {

	const location = "queries.upgrades.legacyOutboxCopier.duplicate"

	count, err := copier.target.CountDocuments(ctx, bson.M{"_id": item.ActivityID})

	if err != nil {
		return derp.Wrap(err, location, "Counting rows", "activityId: "+item.ActivityID.Hex())
	}

	if count > 0 {
		copier.present++
		return nil
	}

	copier.skip(derp.Conflict(location, "Another OutboxItem holds this row's URL", "activityId: "+item.ActivityID.Hex(), "url: "+item.URL))
	return nil
}
