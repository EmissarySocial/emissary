package service

import (
	"context"
	"crypto"
	"iter"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/EmissarySocial/emissary/tools/ascacherules"
	"github.com/EmissarySocial/emissary/tools/ashash"
	"github.com/EmissarySocial/emissary/tools/asnormalizer"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/sherlock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivityStream implements the Hannibal HTTP client interface, and provides a cache for ActivityStream documents.
type ActivityStream struct {
	commonDatabase data.Server   // Database connection for the commonDatabase
	serverFactory  ServerFactory // SessionFactory that creates sessions in domain databases
	factory        *Factory
	hostname       string
	version        string

	actorType string
	actorID   primitive.ObjectID
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// NewActivityStream creates a new ActivityStream service
func NewActivityStream(serverFactory ServerFactory, commonDatabase data.Server, factory *Factory, hostname string, version string, actorType string, actorID primitive.ObjectID) ActivityStream {
	return ActivityStream{
		serverFactory:  serverFactory,
		commonDatabase: commonDatabase,
		factory:        factory,
		hostname:       hostname,
		version:        version,

		actorType: actorType,
		actorID:   actorID,
	}
}

func (service *ActivityStream) Client() streams.Client {

	// Final layer client looks up hashed values within documents
	return ashash.New(service.CacheClient())
}

func (service *ActivityStream) CacheClient() *ascache.Client {

	// Build a new client stack
	sherlockClient := sherlock.NewClient(
		sherlock.WithUserAgent(service.hostname+" /Emissary@v"+service.version+" (https://emissary.social)"),
		sherlock.WithKeyPairFunc(service.KeyPairFunc()),
	)

	// enforce opinionated data formats
	normalizerClient := asnormalizer.New(sherlockClient)

	// Emergency remove context maker. Will re-implement later.
	// compute document context (if missing)
	// contextMakerClient := ascontextmaker.New(normalizerClient, service.commonDatabase)

	// Emergency remove crawler. Will re-implement later.
	// crawler client will load related documents in the background
	/*
		crawlerClient := ascrawler.New(
			service.factory.Queue(),
			contextMakerClient,
			service.actorType,
			service.actorID,
			service.hostname,
		)
	*/

	// apply custom caching rules to documents
	// cacheRulesClient := ascacherules.New(crawlerClient)
	cacheRulesClient := ascacherules.New(normalizerClient)

	// cache data in MongoDB
	cacheClient := ascache.New(
		cacheRulesClient,
		service.factory.Queue(),
		service.commonDatabase,
		service.actorType,
		service.actorID,
		service.hostname,
		ascache.WithIgnoreHeaders(),
	)

	return cacheClient
}

/******************************************
 * Hannibal HTTP Client Interface
 ******************************************/

// Put adds a single document to the ActivityStream cache
func (service *ActivityStream) Save(document streams.Document) error {
	return service.Client().Save(document)
}

// Delete removes a single document from the database by its URL
func (service *ActivityStream) Delete(url string) error {
	return service.Client().Delete(url)
}

/******************************************
 * Custom Query Methods
 ******************************************/

func (service *ActivityStream) Range(ctx context.Context, criteria exp.Expression, options ...option.Option) iter.Seq[ascache.Value] {

	const location = "service.ActivityStream.Range"

	return func(yield func(ascache.Value) bool) {

		// Connect to the database
		collection, err := service.collection(ctx)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to connect to database"))
			return
		}

		// Query the database
		iterator, err := collection.Iterator(criteria, options...)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to query database", criteria))
			return
		}

		// Return results to caller one-by-one
		for value := ascache.NewValue(); iterator.Next(&value); value = ascache.NewValue() {
			if !yield(value) {
				return
			}
		}
	}
}

func (service *ActivityStream) QueryByContext(ctx context.Context, contextName string, afterDate int64, maxRows int) (sliceof.Object[streams.Document], error) {

	// RULE: Do not query empty contexts
	if contextName == "" {
		return sliceof.NewObject[streams.Document](), nil
	}

	// Query the database
	criteria := exp.Equal("object.context", contextName).AndGreaterThan("published", afterDate)
	values := service.Range(ctx, criteria, option.SortAsc("published"), option.MaxRows(int64(maxRows)))
	result := sliceof.NewObject[streams.Document]()

	// Map into model.DocumentLink records
	for value := range values {
		result = append(result, value.AsDocument())
	}

	return result, nil
}

/*
func (service *ActivityStream) QueryByContext_Tree(ctx context.Context, contextName string) (sliceof.Object[*treebuilder.Tree[model.DocumentLink]], error) {

	// RULE: Do not query empty contexts
	if contextName == "" {
		return sliceof.NewObject[*treebuilder.Tree[model.DocumentLink]](), nil
	}

	// Query the database
	criteria := exp.Equal("object.context", contextName)
	values := service.Range(ctx, criteria, option.SortAsc("object.published"))
	treeInput := sliceof.NewObject[model.DocumentLink]()

	// Map into model.DocumentLink records
	for value := range values {
		treeInput = append(treeInput, service.asDocumentLink(value))
	}

	return treebuilder.ParseAndFormat(treeInput), nil
}
*/

func (service *ActivityStream) QueryActors(queryString string) ([]model.ActorSummary, error) {

	const location = "service.ActivityStream.QueryActors"

	// If we think this is an address we can work with (because sherlock says so)
	// the try to retrieve it directly.
	if sherlock.IsValidAddress(queryString) {

		// Try to load the actor directly from the Interwebs
		if newActor, err := service.Client().Load(queryString, sherlock.AsActor()); err == nil {

			// If this is a valid, but (previously) unknown actor, then add it to the results
			// This will also automatically get cached/crawled for next time.
			result := []model.ActorSummary{{
				ID:          newActor.ID(),
				Type:        newActor.Type(),
				Name:        newActor.Name(),
				Icon:        newActor.Icon().Href(),
				Username:    newActor.PreferredUsername(),
				KeyPackages: newActor.KeyPackages().ID(),
			}}

			return result, nil
		}
	}

	// Fall through means that we can't find a perfect match, so fall back to a full-text search
	ctx, cancel := timeoutContext(2)
	defer cancel()

	collection, err := service.collection(ctx)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to connect to database")
	}

	result, err := queries.SearchActivityStreamActors(collection, queryString)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to query database")
	}

	return result, nil
}

// QueryReplies returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStream) QueryReplies(ctx context.Context, inReplyTo string, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation(ctx, "Reply", inReplyTo, "after", 0, done)
}

// QueryRepliesBeforeDate returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStream) QueryRepliesBeforeDate(ctx context.Context, inReplyTo string, maxDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation(ctx, "Reply", inReplyTo, "before", maxDate, done)
}

// QueryRepliesAfterDate returns a slice of streams.Document values that are replies to the specified document, and were published after the specified date.
func (service *ActivityStream) QueryRepliesAfterDate(ctx context.Context, inReplyTo string, minDate int64, maxRows int64) sliceof.Object[ascache.Value] {

	criteria := exp.Equal("metadata.relationType", vocab.RelationTypeReply).
		AndEqual("metadata.relationHref", inReplyTo).
		AndGreaterThan("published", minDate)

	result := sliceof.NewObject[ascache.Value]()

	values := service.Range(
		ctx,
		criteria,
		option.SortAsc("published"),
		option.MaxRows(maxRows),
	)

	for value := range values {
		result = append(result, value)
	}

	return result
}

func (service *ActivityStream) QueryAnnouncesBeforeDate(ctx context.Context, relationHref string, maxDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation(ctx, vocab.ActivityTypeAnnounce, relationHref, "before", maxDate, done)
}

func (service *ActivityStream) QueryLikesBeforeDate(ctx context.Context, relationHref string, maxDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation(ctx, vocab.ActivityTypeLike, relationHref, "before", maxDate, done)
}

// QueryRepliesBeforeDate returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStream) queryByRelation(ctx context.Context, relationType string, relationHref string, cutType string, cutDate int64, done <-chan struct{}) <-chan streams.Document {

	const location = "service.ActivityStream.QueryRelated"

	result := make(chan streams.Document)

	go func() {

		defer close(result)

		// Build the query
		criteria := exp.
			Equal("metadata.relationType", relationType).
			AndEqual("metadata.relationHref", relationHref)

		var sortOption option.Option

		if cutType == "before" {
			criteria = criteria.AndLessThan("object.published", time.Unix(cutDate, 0))
			sortOption = option.SortDesc("object.published")
		} else {
			criteria = criteria.AndGreaterThan("object.published", time.Unix(cutDate, 0))
			sortOption = option.SortAsc("object.published")
		}

		// Try to query the database
		documents, err := service.documentIterator(ctx, criteria, sortOption)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to query database"))
			return
		}

		defer derp.ReportFunc(documents.Close)

		// Write documents into the result channel until done (or done)

		for value := ascache.NewValue(); documents.Next(&value); value = ascache.NewValue() {

			select {

			case <-done:
				return

			default:
				result <- streams.NewDocument(
					value.Object,
					streams.WithHTTPHeader(value.HTTPHeader),
					streams.WithMetadata(value.Metadata),
					streams.WithClient(service.Client()),
				)
			}
		}
	}()

	return result
}

// GetRecipient retrieves the recipient's ID and inbox URL
func (service *ActivityStream) GetRecipient(recipient string) (string, string, error) {

	const location = "service.ActivityStream.GetRecipient"

	// Try to load the recipient as a JSON-LD document
	document, err := service.Client().Load(recipient, sherlock.AsActor())

	if err != nil {
		return "", "", derp.Wrap(err, location, "Unable to load ActivityPub Actor", recipient)
	}

	if !document.IsActor() {
		return "", "", derp.NotFoundError(location, "Recipient is not an ActivityPub Actor", recipient)
	}

	// Successssssssss.
	return document.ID(), document.Inbox().String(), nil
}

/******************************************
 * Custom Actions
 ******************************************/

// SendMessage sends an ActivityPub message to a single recipient/inboxURL
// `inboxURL` the URL to deliver the message to
// `actorType` the type of actor that is sending the message (User, Stream, Search)
// `actorID` unique ID of the actor (zero value for Search Actor)
// `message` the ActivityPub message to send
func (service *ActivityStream) SendMessage(session data.Session, args mapof.Any) error {

	const location = "service.ActivityStream.SendMessage"

	// Collect the Actor to receive the message
	recipientID := args.GetString("to")

	if recipientID == "" {
		return derp.NotFoundError(location, "Recipient ID is required", recipientID)
	}

	// Collect the message to be sent
	message := args.GetMap("message")

	if message.IsEmpty() {
		return derp.NotFoundError(location, "Message is required", message)
	}

	// Find ActivityPub Actor
	locatorService := service.factory.Locator()
	actor, err := locatorService.GetActor(session, args.GetString("actorType"), args.GetString("actorID"))

	if err != nil {
		return derp.Wrap(err, location, "Unable to find ActivityPub Actor")
	}

	// Send the message to the recipientID
	if err := actor.SendOne(recipientID, message); err != nil {
		return derp.Wrap(err, location, "Unable to send message", message, derp.WithInternalError())
	}

	// Success!!
	return nil
}

func (service *ActivityStream) PublicKeyFinder(keyID string) (string, error) {

	const location = "service.ActivityStream.PublicKeyFinder"

	actorID, _, _ := strings.Cut(keyID, "#")

	actor := streams.NewDocument(mapof.Any{
		vocab.PropertyID: actorID,
	})

	// Load the Actor from the document
	actor, err := actor.Load(sherlock.AsActor())

	if err != nil {
		return "", derp.Wrap(err, location, "Error retrieving Actor from ActivityPub document", actor.Value())
	}

	// Search the Actor's public keys for the one that matches the provided keyID
	for key := range actor.PublicKey().Range() {

		if key.ID() == keyID {
			return key.PublicKeyPEM(), nil
		}
	}

	return "", derp.NotFoundError(location, "Public Key not found", keyID)
}

// KeyPairFunc returns a function that will locate the public/private key pair
// for the specidied URL.  This can only be used for local URLs
func (service *ActivityStream) KeyPairFunc() sherlock.KeyPairFunc {

	const location = "service.ActivityStream.KeyPairFunc"

	return func() (string, crypto.PrivateKey) {

		// Get the Domain Factory
		domainFactory, err := service.serverFactory.ByHostname(service.hostname)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Invalid hostname. No database found."))
			return "", nil
		}

		session, cancel, err := domainFactory.Session(10 * time.Second)
		defer cancel()

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to connect to database"))
			return "", nil
		}

		// USE service.actorType and service.actorID to retrieve the required PEM keys.
		locatorService := domainFactory.Locator()
		publicKeyID, privateKey, err := locatorService.GetPrivateKey(session, service.actorType, service.actorID)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to retrieve private key"))
			return "", nil
		}

		return publicKeyID, privateKey
	}
}

/******************************************
 * Helper Methods
 ******************************************/

// iterator reads from the database and returns a data.Iterator with the result values.
func (service *ActivityStream) documentIterator(ctx context.Context, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {

	const location = "service.ActivityStream.documentIterator"

	// Forward request to collection
	collection, err := service.collection(ctx)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to query database", criteria)
	}

	if collection == nil {
		return nil, derp.InternalError(location, "Collection cannot be nil. This should never happen.")
	}

	return collection.Iterator(criteria, options...)
}

// collection creates a new mongodb Session and returns the mongodb Collection that stores ActivityStream documents
func (service *ActivityStream) collection(ctx context.Context) (data.Collection, error) {

	const location = "service.ActivityStream.collection"

	// NILCHECK: commonDatabase must be populated
	if service.commonDatabase == nil {
		return nil, derp.InternalError(location, "Service not initialized")
	}

	// Connect to the database
	session, err := service.commonDatabase.Session(ctx)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to connect to database")
	}

	// NILCHECK: session cannot be nil.
	if session == nil {
		return nil, derp.InternalError(location, "Database session is nil. This should never happen.")
	}

	// Return the collection
	return session.Collection("Document"), nil
}

/*
func (service *ActivityStream) asDocumentLink(value ascache.Value) model.DocumentLink {

	document := streams.NewDocument(value.Object)
	attributedTo := document.AttributedTo()

	return model.DocumentLink{
		ID:        document.ID(),
		InReplyTo: document.InReplyTo().ID(),
		Name:      document.Name(),
		Icon:      document.Icon().Href(),
		Summary:   document.Summary(),
		Content:   document.Content(),
		AttributedTo: model.PersonLink{
			Username:   attributedTo.PreferredUsername(),
			ProfileURL: attributedTo.ID(),
			Name:       attributedTo.Name(),
			IconURL:    attributedTo.Icon().Href(),
		},
		Published: document.Published().Unix(),
		Token:     value.Metadata.HashedID,
	}
}
*/
