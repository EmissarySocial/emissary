//go:build replycontrols

// Command rchelper does the database, JSON, and polling work for run.sh.  The build tag keeps it out
// of `go build ./...` and `go test ./...`; run.sh builds it with `-tags replycontrols`.
package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// publicIRI is the only spelling of Public that GoToSocial accepts in an approval list
const publicIRI = "https://www.w3.org/ns/activitystreams#Public"

// main dispatches one subcommand, and exits 1 with a message on any failure
func main() {

	if len(os.Args) < 2 {
		fail("usage: rchelper <command> [args...]")
	}

	// hannibal traces every signature it makes, which would bury run.sh's own output
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	command, args := os.Args[1], os.Args[2:]

	switch command {
	case "freeport":
		freePort()
	case "list-dbs":
		need(args, 1)
		listDatabases(args[0])
	case "drop-db":
		need(args, 1)
		dropDatabase(args[0])
	case "latest-stream":
		need(args, 2)
		latestStream(args[0], args[1])
	case "set-reply-policy":
		need(args, 3)
		setReplyPolicy(args[0], args[1], args[2])
	case "reply-count":
		need(args, 2)
		replyCount(args[0], args[1])
	case "count-notifications":
		need(args, 3)
		countNotifications(args[0], args[1], args[2])
	case "wait-inbox":
		need(args, 3)
		waitInbox(args[0], args[1], seconds(args[2]))
	case "wait-queue-idle":
		need(args, 2)
		waitQueueIdle(args[0], seconds(args[1]))
	case "check-policy":
		need(args, 2)
		checkPolicy(args[0], args[1])
	case "replies":
		need(args, 1)
		replies(args[0])
	case "diagnose-cache":
		need(args, 3)
		diagnoseCache(args[0], args[1], args[2])
	case "quote-request":
		need(args, 5)
		quoteRequest(args[0], args[1], args[2], args[3], args[4])
	case "accept-result":
		need(args, 2)
		acceptResult(args[0], args[1])
	case "count-inbox":
		need(args, 2)
		countInbox(args[0], args[1])
	case "check-authorization":
		need(args, 4)
		checkAuthorization(args[0], args[1], args[2], args[3])
	default:
		fail("unknown command: " + command)
	}
}

/******************************************
 * Process helpers
 ******************************************/

// need exits unless at least count arguments are present
func need(args []string, count int) {
	if len(args) < count {
		fail("expected " + strconv.Itoa(count) + " arguments")
	}
}

// seconds parses a timeout in whole seconds
func seconds(value string) time.Duration {
	result, err := strconv.Atoi(value)
	if err != nil {
		fail("invalid timeout: " + value)
	}
	return time.Duration(result) * time.Second
}

// fail prints the message to stderr and exits 1
func fail(message string) {
	fmt.Fprintln(os.Stderr, "rchelper: "+message)
	os.Exit(1)
}

/******************************************
 * MongoDB helpers
 ******************************************/

// listDatabases prints every database name that starts with the prefix, one per line
func listDatabases(prefix string) {

	requirePrefix(prefix)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	names, err := client.ListDatabaseNames(ctx, bson.M{"name": bson.M{"$regex": "^" + prefix}})

	if err != nil {
		fail("listing databases: " + err.Error())
	}

	for _, name := range names {
		fmt.Println(name)
	}
}

// dropDatabase drops one rctest_ database
func dropDatabase(database string) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	if err := client.Database(database).Drop(ctx); err != nil {
		fail("dropping " + database + ": " + err.Error())
	}
}

// latestStream prints the ID of the newest Stream with the given templateId
func latestStream(database string, templateID string) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	var result struct {
		ID primitive.ObjectID `bson:"_id"`
	}

	opts := options.FindOne().SetSort(bson.M{"_id": -1})
	if err := client.Database(database).Collection("Stream").FindOne(ctx, bson.M{"templateId": templateID}, opts).Decode(&result); err != nil {
		fail("finding the newest " + templateID + " Stream: " + err.Error())
	}

	fmt.Println(result.ID.Hex())
}

// setReplyPolicy writes Stream.ReplyPolicy directly, because no embedded template offers the toggle
func setReplyPolicy(database string, streamID string, policy string) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	update := bson.M{"$set": bson.M{"replyPolicy": policy}}
	result, err := client.Database(database).Collection("Stream").UpdateOne(ctx, bson.M{"_id": objectID(streamID)}, update)

	if err != nil {
		fail("setting replyPolicy: " + err.Error())
	}

	if result.MatchedCount != 1 {
		fail("no Stream " + streamID)
	}
}

// replyCount prints the stored Stream.ReplyCount
func replyCount(database string, streamID string) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	var result struct {
		ReplyCount int `bson:"replyCount"`
	}

	if err := client.Database(database).Collection("Stream").FindOne(ctx, bson.M{"_id": objectID(streamID)}).Decode(&result); err != nil {
		fail("loading Stream " + streamID + ": " + err.Error())
	}

	fmt.Println(result.ReplyCount)
}

// objectID parses a hex ObjectID
func objectID(value string) primitive.ObjectID {
	result, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		fail("invalid ObjectID: " + value)
	}
	return result
}

// countNotifications prints how many live Notifications of the type name the object URL
func countNotifications(database string, notificationType string, objectURL string) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	filter := bson.M{"type": notificationType, "objectUrl": objectURL, "deleteDate": 0}
	count, err := client.Database(database).Collection("Notification").CountDocuments(ctx, filter)

	if err != nil {
		fail("counting Notifications: " + err.Error())
	}

	fmt.Println(count)
}

// waitInbox waits until the domain's Inbox records an activity whose object is the URL, which proves delivery
func waitInbox(database string, objectURL string, timeout time.Duration) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	collection := client.Database(database).Collection("Inbox")

	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); time.Sleep(250 * time.Millisecond) {
		if count, err := collection.CountDocuments(ctx, bson.M{"objectId": objectURL}); err == nil && count > 0 {
			return
		}
	}

	fail("timed out waiting for " + objectURL + " in " + database + ".Inbox")
}

// waitQueueIdle waits until the shared queue holds nothing but its own schedule tasks
func waitQueueIdle(database string, timeout time.Duration) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	collection := client.Database(database).Collection("Queue")
	filter := bson.M{"name": bson.M{"$nin": bson.A{"ScheduleDaily", "ScheduleHourly"}}}

	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); time.Sleep(250 * time.Millisecond) {
		if count, err := collection.CountDocuments(ctx, filter); err == nil && count == 0 {
			return
		}
	}

	fail("timed out waiting for the queue in " + database + " to drain")
}

// acceptResult prints the result of the Accept that answered the request, as stored in the requester's Inbox
func acceptResult(database string, requestID string) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	var result struct {
		RawActivity mapof.Any `bson:"rawActivity"`
	}

	filter := bson.M{"activityType": vocab.ActivityTypeAccept, "objectId": requestID}
	if err := client.Database(database).Collection("Inbox").FindOne(ctx, filter).Decode(&result); err != nil {
		fail("loading the Accept of " + requestID + ": " + err.Error())
	}

	fmt.Println(result.RawActivity.GetString(vocab.PropertyResult))
}

// countInbox prints how many activities in the domain's Inbox name the object, which proves an absence
func countInbox(database string, objectURL string) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	count, err := client.Database(database).Collection("Inbox").CountDocuments(ctx, bson.M{"objectId": objectURL})

	if err != nil {
		fail("counting Inbox activities: " + err.Error())
	}

	fmt.Println(count)
}

// requirePrefix refuses any database this harness did not name
func requirePrefix(database string) {
	if !strings.HasPrefix(database, "rctest_") {
		fail("refusing to touch a database outside the rctest_ prefix: " + database)
	}
}

// connect opens a client to the test replica set
func connect() (*mongo.Client, context.Context) {

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI()).SetServerSelectionTimeout(5*time.Second))

	if err != nil {
		fail("connecting to MongoDB: " + err.Error())
	}

	return client, ctx
}

// mongoURI returns the test replica set's connection string, overridable with RC_MONGO_URI
func mongoURI() string {
	if uri := os.Getenv("RC_MONGO_URI"); uri != "" {
		return uri
	}
	return "mongodb://127.0.0.1:27017/?directConnection=true"
}

/******************************************
 * ActivityPub helpers
 ******************************************/

// checkPolicy verifies the published JSON-LD: canReply and canQuote each approve exactly the expected
// IRI, the gts: terms are declared, and commentsEnabled is absent.  It prints the evidence either way.
func checkPolicy(url string, expected string) {

	if expected == "public" {
		expected = publicIRI
	}

	document := getJSON(url)
	problems := make([]string, 0)
	evidence := make([]string, 0, 2)

	// Each automaticApproval must be exactly [expected], because anyone who may reply may also quote
	policy, _ := document["interactionPolicy"].(map[string]any)

	for _, permission := range []string{"canReply", "canQuote"} {
		subPolicy, _ := policy[permission].(map[string]any)
		approval, _ := subPolicy["automaticApproval"].([]any)
		approvalJSON, _ := json.Marshal(subPolicy["automaticApproval"])
		evidence = append(evidence, permission+"="+string(approvalJSON))

		if (len(approval) != 1) || (approval[0] != expected) {
			problems = append(problems, permission+".automaticApproval is "+string(approvalJSON)+", want [\""+expected+"\"]")
		}

		if _, exists := subPolicy["manualApproval"]; exists {
			problems = append(problems, permission+".manualApproval is published")
		}
	}

	// The @context must declare gts: and the terms it uses
	contextJSON, _ := json.Marshal(document["@context"])
	for _, term := range []string{`"gts":"https://gotosocial.org/ns#"`, `"interactionPolicy":{"@id":"gts:interactionPolicy"`, `"canReply":{"@id":"gts:canReply"`, `"canQuote":{"@id":"gts:canQuote"`, `"automaticApproval":{"@id":"gts:automaticApproval"`} {
		if !strings.Contains(string(contextJSON), term) {
			problems = append(problems, "@context lacks "+term)
		}
	}

	// PeerTube's vocabulary was removed on 2026-10-22
	if _, exists := document["commentsEnabled"]; exists {
		problems = append(problems, "commentsEnabled is published")
	}

	if strings.Contains(string(contextJSON), "commentsEnabled") {
		problems = append(problems, "@context declares commentsEnabled")
	}

	if len(problems) > 0 {
		fail(strings.Join(problems, "; "))
	}

	fmt.Println(strings.Join(evidence, ", ") + ", gts terms declared, no commentsEnabled")
}

// replies prints a Replies collection as "<totalItems> <item> <item>..."
func replies(url string) {

	collection := getJSON(url)
	items := make([]string, 0)

	if ordered, ok := collection["orderedItems"].([]any); ok {
		for _, item := range ordered {
			items = append(items, streams.NewDocument(item).ID())
		}
	}

	total := "missing"
	if value, ok := collection["totalItems"].(float64); ok {
		total = strconv.Itoa(int(value))
	}

	fmt.Println(strings.TrimSpace(total + " " + strings.Join(items, " ")))
}

// diagnoseCache evaluates canReply for the actor twice, on the live JSON and on the copy in the
// shared ActivityStream cache, and prints both with the Go type of the cached approval list
func diagnoseCache(database string, url string, actorID string) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	cached := ascache.NewValue()
	if err := client.Database(database).Collection("Document").FindOne(ctx, bson.M{"urls": url}).Decode(&cached); err != nil {
		fail("loading cached " + url + ": " + err.Error())
	}

	live := streams.NewDocument(getJSON(url))
	fromCache := streams.NewDocument(cached.Object)

	approvalType := "missing"
	if policy, ok := cached.Object["interactionPolicy"].(mapof.Any); ok {
		if canReply, ok := policy["canReply"].(mapof.Any); ok {
			approvalType = fmt.Sprintf("%T", canReply["automaticApproval"])
		}
	}

	fmt.Printf("live=%s cached=%s cachedAutomaticApprovalType=%s\n",
		canReplyResult(live, actorID), canReplyResult(fromCache, actorID), approvalType)
}

// canReplyResult names hannibal's canReply answer for the actor on the document
func canReplyResult(document streams.Document, actorID string) string {

	request := streams.InteractionRequest{
		ActorID:   actorID,
		AuthorID:  document.AttributedTo().ID(),
		Mentioned: slices.Collect(document.RangeMentions()),
		IsMember:  func(string) bool { return false },
	}

	switch document.InteractionPolicy().CanInteract(vocab.PropertyCanReply, request) {
	case streams.InteractionAllowed:
		return "Allowed"
	case streams.InteractionNeedsApproval:
		return "NeedsApproval"
	default:
		return "Denied"
	}
}

// quoteRequest signs a FEP-044f QuoteRequest with the actor's own key, read from the actor's database, and
// POSTs it to the quoted note's author.  The quote post is inlined, as Mastodon does, and attributed to
// attributedTo.  It prints the HTTP status of the delivery.
func quoteRequest(database string, actorID string, noteURL string, quoteID string, attributedTo string) {

	requirePrefix(database)
	client, ctx := connect()
	defer client.Disconnect(ctx) //nolint:errcheck

	// Load the actor's private key.  Each domain here has one User, so it has one User key
	var key struct {
		PrivatePEM string `bson:"privatePEM"`
	}

	if err := client.Database(database).Collection("EncryptionKey").FindOne(ctx, bson.M{"parentType": "User", "deleteDate": 0}).Decode(&key); err != nil {
		fail("loading the actor's key: " + err.Error())
	}

	block, _ := pem.Decode([]byte(key.PrivatePEM))
	if block == nil {
		fail("decoding the actor's key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fail("parsing the actor's key: " + err.Error())
	}

	// Find the key's public ID, and the inbox of the note's author
	keyID := streams.NewDocument(getJSON(actorID)).PublicKey().ID()
	authorID := streams.NewDocument(getJSON(noteURL)).AttributedTo().ID()
	inboxURL := streams.NewDocument(getJSON(authorID)).Inbox().ID()

	if (keyID == "") || (inboxURL == "") {
		fail("finding the key ID of " + actorID + " or the inbox of " + authorID)
	}

	// Build the QuoteRequest, with the quote post inlined
	body, err := json.Marshal(mapof.Any{
		vocab.AtContext:      []any{vocab.NamespaceActivityStreams, vocab.ContextTypeQuotePosts},
		vocab.PropertyID:     quoteID + "#quote_request",
		vocab.PropertyType:   vocab.ActivityTypeQuoteRequest,
		vocab.PropertyActor:  actorID,
		vocab.PropertyObject: noteURL,
		vocab.PropertyInstrument: mapof.Any{
			vocab.PropertyID:           quoteID,
			vocab.PropertyType:         vocab.ObjectTypeNote,
			vocab.PropertyAttributedTo: attributedTo,
			vocab.PropertyContent:      "<p>Quoting " + noteURL + "</p>",
			vocab.PropertyQuote:        noteURL,
			vocab.PropertyTo:           []string{publicIRI},
		},
	})

	if err != nil {
		fail("encoding the QuoteRequest: " + err.Error())
	}

	// Sign and deliver it
	request, err := http.NewRequest(http.MethodPost, inboxURL, bytes.NewReader(body))
	if err != nil {
		fail("building the request: " + err.Error())
	}

	request.Header.Set("Content-Type", "application/activity+json")
	request.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))

	if err := sigs.Sign(request, keyID, privateKey); err != nil {
		fail("signing the QuoteRequest: " + err.Error())
	}

	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		fail("POST " + inboxURL + ": " + err.Error())
	}

	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	fmt.Println(response.StatusCode)
}

// checkAuthorization verifies a QuoteAuthorization the way Mastodon's verify_quote_service.rb does, and
// prints the evidence either way
func checkAuthorization(url string, noteURL string, quoteID string, authorID string) {

	document := getJSON(url)
	problems := make([]string, 0)

	expected := map[string]string{
		vocab.PropertyID:                url,
		vocab.PropertyType:              vocab.ObjectTypeQuoteAuthorization,
		vocab.PropertyAttributedTo:      authorID,
		vocab.PropertyInteractingObject: quoteID,
		vocab.PropertyInteractionTarget: noteURL,
	}

	for property, value := range expected {
		if got, _ := document[property].(string); got != value {
			problems = append(problems, property+" is \""+got+"\", want \""+value+"\"")
		}
	}

	// The @context must declare the type and the gts: terms
	contextJSON, _ := json.Marshal(document["@context"])
	for _, term := range []string{`"QuoteAuthorization":"https://w3id.org/fep/044f#QuoteAuthorization"`, `"interactingObject":{"@id":"gts:interactingObject"`, `"interactionTarget":{"@id":"gts:interactionTarget"`} {
		if !strings.Contains(string(contextJSON), term) {
			problems = append(problems, "@context lacks "+term)
		}
	}

	if len(problems) > 0 {
		fail(strings.Join(problems, "; "))
	}

	fmt.Println("interactingObject=" + quoteID + ", interactionTarget=" + noteURL + ", attributedTo=" + authorID + ", terms declared")
}

// getJSON fetches an ActivityPub document as a map
func getJSON(url string) mapof.Any {

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fail("building request: " + err.Error())
	}

	request.Header.Set("Accept", "application/activity+json")
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)

	if err != nil {
		fail("GET " + url + ": " + err.Error())
	}

	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)

	if response.StatusCode != http.StatusOK {
		fail("GET " + url + ": HTTP " + strconv.Itoa(response.StatusCode))
	}

	result := mapof.NewAny()
	if err := json.Unmarshal(body, &result); err != nil {
		fail("decoding " + url + ": " + err.Error())
	}

	return result
}

/******************************************
 * Network helpers
 ******************************************/

// freePort prints a TCP port that is free on 127.0.0.1 right now
func freePort() {

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fail("finding a free port: " + err.Error())
	}

	defer listener.Close()
	fmt.Println(listener.Addr().(*net.TCPAddr).Port)
}
