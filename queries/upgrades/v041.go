package upgrades

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/******************************************
 * Version 41 moves every Stream to the "isOutbox" flag
 *
 * A Stream that its author's outbox already holds is
 * flagged, and marked as held by followers.  The old
 * publish dates are cleared, and the Webhook collection
 * is dropped.  Every step is safe to run more than once.
 ******************************************/

// version41BatchSize is the number of Streams that one bulk write changes
const version41BatchSize = 500

// version41Move names the State that an unsent Stream of one Template leaves, and the State it
// moves to. An upgrade cannot read Templates, so each move is written out by Template ID.
type version41Move struct {
	from             string // State that says more than is true of an unsent Stream
	to               string // State that only the author can view
	unlessParentFrom string // The Stream stays when its parent is in this State
}

// version41Moves lists the Templates whose published State was split in two
var version41Moves = map[string]version41Move{

	// A song was sent only when its album was published, and shared one State either way
	"bandwagon-song": {from: "published", to: "default", unlessParentFrom: "published"},
}

// Version41Count counts the Streams of one Template in one State
type Version41Count struct {
	TemplateID string // Template of the Streams counted
	StateID    string // State of the Streams counted, before any move
	Total      int    // Streams that are not deleted
	Held       int    // Streams whose author's outbox holds a Create or an Update of them
	Flagged    int    // Held Streams that this upgrade flags
	Moved      int    // Unsent Streams that this upgrade moves to another State
	Hidden     int    // Streams that the old publish dates were hiding
}

// Version41Plan is everything that Version41 would change in the Streams of one database
type Version41Plan struct {
	Counts []Version41Count // One entry per Template and State, in order
	flag   []primitive.ObjectID
	move   map[string][]primitive.ObjectID
}

// version41Stream holds the properties of a stored Stream that the plan reads
type version41Stream struct {
	StreamID      primitive.ObjectID `bson:"_id"`
	ParentID      primitive.ObjectID `bson:"parentId"`
	TemplateID    string             `bson:"templateId"`
	StateID       string             `bson:"stateId"`
	URL           string             `bson:"url"`
	Fingerprint   string             `bson:"fingerprint"`
	PublishDate   int64              `bson:"publishDate"`
	UnPublishDate *int64             `bson:"unpublishDate"`
	AttributedTo  struct {
		UserID primitive.ObjectID `bson:"userId"`
	} `bson:"attributedTo"`
}

// Version41 flags every Stream that followers already hold, moves the unsent Streams of a split
// State, clears the old publish dates, and drops the Webhook collection
func Version41(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version41"

	fmt.Println("... Version 41")

	// Report before writing
	plan, err := PlanVersion41(ctx, session)

	if err != nil {
		return derp.Wrap(err, location, "Planning the upgrade")
	}

	plan.report(session.Name())

	streams := session.Collection("Stream")

	// RULE: The flag and the placeholder are written together, and only where followers are
	// not yet recorded as holding anything, so a fingerprint that the outbox task wrote stands
	held := bson.M{"$set": bson.M{"isOutbox": true, "fingerprint": model.StreamFingerprintPending}}

	if err := version41Update(ctx, streams, plan.flag, version41Unrecorded(), held); err != nil {
		return derp.Wrap(err, location, "Flagging Streams")
	}

	for templateID, streamIDs := range plan.move {
		move := version41Moves[templateID]
		filter := bson.M{"templateId": templateID, "stateId": move.from}
		update := bson.M{"$set": bson.M{"stateId": move.to}}

		if err := version41Update(ctx, streams, streamIDs, filter, update); err != nil {
			return derp.Wrap(err, location, "Moving Streams", templateID)
		}
	}

	// The flag is always stored, so that a query can match FALSE
	missing := bson.M{"isOutbox": bson.M{"$exists": false}}

	if _, err := streams.UpdateMany(ctx, missing, bson.M{"$set": bson.M{"isOutbox": false}}); err != nil {
		return derp.Wrap(err, location, "Storing the flag on every other Stream")
	}

	// Both collections store a model.Stream, so a draft holds the same two dates
	for _, name := range []string{"Stream", "StreamDraft"} {
		if err := version41ClearDates(ctx, session.Collection(name)); err != nil {
			return derp.Wrap(err, location, "Clearing publish dates", name)
		}
	}

	// Outbound webhooks are gone.  Dropping a collection that does not exist is not an error.
	if err := session.Collection("Webhook").Drop(ctx); err != nil {
		return derp.Wrap(err, location, "Dropping the Webhook collection")
	}

	// The flag is planted
	return nil
}

// PlanVersion41 reads a database and returns what Version41 would change in its Streams. It
// writes nothing, so it is also the upgrade's dry run.
func PlanVersion41(ctx context.Context, session *mongo.Database) (Version41Plan, error) {

	const location = "queries.upgrades.PlanVersion41"

	result := Version41Plan{move: make(map[string][]primitive.ObjectID)}

	held, err := version41HeldObjects(ctx, session)

	if err != nil {
		return result, derp.Wrap(err, location, "Reading the outboxes")
	}

	streams, err := version41Streams(ctx, session.Collection("Stream"))

	if err != nil {
		return result, derp.Wrap(err, location, "Reading Streams")
	}

	// A move depends on the State of the Stream's parent
	states := make(map[primitive.ObjectID]string, len(streams))

	for _, stream := range streams {
		states[stream.StreamID] = stream.StateID
	}

	counts := make(map[[2]string]*Version41Count)
	now := time.Now().Unix()

	for _, stream := range streams {

		key := [2]string{stream.TemplateID, stream.StateID}
		count, exists := counts[key]

		if !exists {
			count = &Version41Count{TemplateID: stream.TemplateID, StateID: stream.StateID}
			counts[key] = count
		}

		count.Total++

		if stream.hiddenByDates(now) {
			count.Hidden++
		}

		// RULE: The flag comes from the outbox rows, never from the name of the State
		if stream.URL != "" && held[version41HeldKey(stream.AttributedTo.UserID, stream.URL)] {
			count.Held++

			if stream.Fingerprint == "" {
				count.Flagged++
				result.flag = append(result.flag, stream.StreamID)
			}

			continue
		}

		if move, exists := version41Moves[stream.TemplateID]; exists {
			if move.applies(stream.StateID, states[stream.ParentID]) {
				count.Moved++
				result.move[stream.TemplateID] = append(result.move[stream.TemplateID], stream.StreamID)
			}
		}
	}

	// Sort the counts, so that two runs over the same data report the same lines
	for _, count := range counts {
		result.Counts = append(result.Counts, *count)
	}

	slices.SortFunc(result.Counts, func(a Version41Count, b Version41Count) int {
		if a.TemplateID != b.TemplateID {
			if a.TemplateID < b.TemplateID {
				return -1
			}
			return 1
		}

		if a.StateID < b.StateID {
			return -1
		}

		if a.StateID > b.StateID {
			return 1
		}

		return 0
	})

	return result, nil
}

// applies returns TRUE when an unsent Stream in this State, beneath a parent in that State, is moved
func (move version41Move) applies(stateID string, parentStateID string) bool {

	if stateID != move.from {
		return false
	}

	// RULE: A Stream whose parent is still on the website stays where visitors can see it
	return parentStateID != move.unlessParentFrom
}

// hiddenByDates returns TRUE when the old publish dates kept this Stream out of public lists
func (stream version41Stream) hiddenByDates(now int64) bool {

	if stream.PublishDate > now {
		return true
	}

	return (stream.UnPublishDate != nil) && (*stream.UnPublishDate <= now)
}

// report writes the plan to the log, one line per Template and State
func (plan Version41Plan) report(database string) {

	for _, count := range plan.Counts {
		log.Info().
			Str("database", database).
			Str("template", count.TemplateID).
			Str("state", count.StateID).
			Int("streams", count.Total).
			Int("held", count.Held).
			Int("flagged", count.Flagged).
			Int("moved", count.Moved).
			Int("hiddenByDates", count.Hidden).
			Msg("Moving Streams to the outbox flag")
	}
}

// version41HeldKey names one object in one User's outbox
func version41HeldKey(userID primitive.ObjectID, objectID string) string {
	return userID.Hex() + " " + objectID
}

// version41HeldObjects returns the objects that each User's outbox holds a Create or an Update of
func version41HeldObjects(ctx context.Context, session *mongo.Database) (map[string]bool, error) {

	const location = "queries.upgrades.version41HeldObjects"

	result := make(map[string]bool)

	// RULE: Only a Create or an Update says that followers hold the object, because unpublishing
	// stores a Delete under the same object.  A deleted row is in no outbox.
	filter := bson.M{
		"actorType":    model.ActorTypeUser,
		"activityType": bson.M{"$in": bson.A{vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate}},
		"deleteDate":   bson.M{"$not": bson.M{"$gt": 0}},
	}

	findOptions := options.Find().SetProjection(bson.M{"actorId": 1, "objectId": 1})

	// Slot 39 copied the legacy Outbox into OutboxItem, and the index sync moves Outbox2 there
	// at each start.  All three are read, so that a row either one left behind still counts.
	for _, name := range []string{"OutboxItem", "Outbox2", "Outbox"} {

		if err := version41ReadHeld(ctx, session.Collection(name), filter, findOptions, result); err != nil {
			return nil, derp.Wrap(err, location, "Reading outbox rows", name)
		}
	}

	return result, nil
}

// version41ReadHeld adds the objects named by one collection's matching rows to the set
func version41ReadHeld(ctx context.Context, collection *mongo.Collection, filter bson.M, findOptions *options.FindOptions, held map[string]bool) error {

	const location = "queries.upgrades.version41ReadHeld"

	cursor, err := collection.Find(ctx, filter, findOptions)

	if err != nil {
		return derp.Wrap(err, location, "Finding rows")
	}

	// A failed Close leaves the server to time the cursor out, which costs nothing here
	defer func() { _ = cursor.Close(context.WithoutCancel(ctx)) }()

	for cursor.Next(ctx) {

		row := struct {
			ActorID  primitive.ObjectID `bson:"actorId"`
			ObjectID string             `bson:"objectId"`
		}{}

		// A row that cannot be read names no object, so it is left out
		if err := cursor.Decode(&row); err != nil {
			continue
		}

		if row.ObjectID != "" {
			held[version41HeldKey(row.ActorID, row.ObjectID)] = true
		}
	}

	if err := cursor.Err(); err != nil {
		return derp.Wrap(err, location, "Reading rows")
	}

	return nil
}

// version41Streams returns the properties that the plan reads, for every Stream that is not deleted
func version41Streams(ctx context.Context, collection *mongo.Collection) ([]version41Stream, error) {

	const location = "queries.upgrades.version41Streams"

	// RULE: A deleted Stream is left alone.  Its Delete was sent, or not, before this upgrade.
	filter := bson.M{"deleteDate": bson.M{"$not": bson.M{"$gt": 0}}}

	findOptions := options.Find().SetProjection(bson.M{
		"parentId":            1,
		"templateId":          1,
		"stateId":             1,
		"url":                 1,
		"fingerprint":         1,
		"publishDate":         1,
		"unpublishDate":       1,
		"attributedTo.userId": 1,
	})

	cursor, err := collection.Find(ctx, filter, findOptions)

	if err != nil {
		return nil, derp.Wrap(err, location, "Finding Streams")
	}

	// A failed Close leaves the server to time the cursor out, which costs nothing here
	defer func() { _ = cursor.Close(context.WithoutCancel(ctx)) }()

	result := make([]version41Stream, 0)

	for cursor.Next(ctx) {

		stream := version41Stream{}

		// A Stream that cannot be read is reported and left as it is
		if err := cursor.Decode(&stream); err != nil {
			derp.Report(derp.Wrap(err, location, "Decoding Stream", "_id: "+cursor.Current.Lookup("_id").String()))
			continue
		}

		result = append(result, stream)
	}

	if err := cursor.Err(); err != nil {
		return nil, derp.Wrap(err, location, "Reading Streams")
	}

	return result, nil
}

// version41Unrecorded matches a Stream that is not deleted, and that followers are not yet
// recorded as holding
func version41Unrecorded() bson.M {
	return bson.M{
		"deleteDate":  bson.M{"$not": bson.M{"$gt": 0}},
		"fingerprint": bson.M{"$in": bson.A{nil, ""}},
	}
}

// version41Update applies one update to the listed Streams that still match the filter, a batch at a time
func version41Update(ctx context.Context, collection *mongo.Collection, streamIDs []primitive.ObjectID, filter bson.M, update bson.M) error {

	const location = "queries.upgrades.version41Update"

	for batch := range slices.Chunk(streamIDs, version41BatchSize) {

		// RULE: The filter is checked again by the database, because a Stream may have been
		// saved since the plan read it
		criteria := bson.M{"_id": bson.M{"$in": batch}}

		for key, value := range filter {
			criteria[key] = value
		}

		if _, err := collection.UpdateMany(ctx, criteria, update); err != nil {
			return derp.Wrap(err, location, "Updating Streams")
		}
	}

	return nil
}

// version41ClearDates turns the end-of-time publish date into "no date", and removes the
// unpublish date from every record
func version41ClearDates(ctx context.Context, collection *mongo.Collection) error {

	const location = "queries.upgrades.version41ClearDates"

	sentinel := bson.M{"publishDate": int64(math.MaxInt64)}

	if _, err := collection.UpdateMany(ctx, sentinel, bson.M{"$set": bson.M{"publishDate": int64(0)}}); err != nil {
		return derp.Wrap(err, location, "Clearing publishDate")
	}

	present := bson.M{"unpublishDate": bson.M{"$exists": true}}

	if _, err := collection.UpdateMany(ctx, present, bson.M{"$unset": bson.M{"unpublishDate": ""}}); err != nil {
		return derp.Wrap(err, location, "Removing unpublishDate")
	}

	return nil
}
