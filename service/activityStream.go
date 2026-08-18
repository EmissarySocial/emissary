package service

import (
	"context"
	"crypto"
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/EmissarySocial/emissary/tools/ascacherules"
	"github.com/EmissarySocial/emissary/tools/ashash"
	"github.com/EmissarySocial/emissary/tools/asnormalizer"
	"github.com/EmissarySocial/emissary/tools/asrules"
	"github.com/EmissarySocial/emissary/tools/assanitizer"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/metadata"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/sherlock"
	"github.com/benpate/sherlock/activitypub"
	"github.com/benpate/sherlock/bridgyfed"
	"github.com/benpate/sherlock/tagspub"
	"github.com/benpate/sherlock/tombstone"
	"github.com/benpate/sherlock/webfinger"
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivityStream implements the Hannibal HTTP client interface, and provides a cache for ActivityStream documents.
type ActivityStream struct {
	getCommonDatabase func() data.Server // read LIVE on every use (never captured): a config reload can reconnect the common database, and a captured handle would fail every call with "client is disconnected"
	locatorService    *Locator
	ruleService       *Rule
	hostname          string
	queue             *queue.Queue
	version           string
	newSession        func(time.Duration) (data.Session, context.CancelFunc, error)
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// NewActivityStream creates a new ActivityStream service
func NewActivityStream() ActivityStream {
	return ActivityStream{}
}

// Refresh updates links to additional services that may not have been initialized when this service was created.
func (service *ActivityStream) Refresh(factory *Factory) {
	// The common database is stored as a GETTER, not a value: factory.CommonDatabase() reads
	// through to the server factory, so every call here sees the current connection even after a
	// config reload swaps it.  (The closure over `factory` is safe: a domain's *Factory pointer
	// is stable for its whole lifetime -- reloads Refresh it in place.)
	service.getCommonDatabase = func() data.Server { return factory.CommonDatabase() }
	service.locatorService = factory.Locator()
	service.ruleService = factory.Rule()
	service.hostname = factory.Hostname()
	service.version = factory.Version()
	service.queue = factory.Queue()
	service.newSession = factory.Session
}

// AppClient returns a streams.Client that is configured for the Application actor.
func (service *ActivityStream) AppClient() streams.Client {
	return service.Client(model.ActorTypeApplication, primitive.NilObjectID)
}

// UserClient returns a streams.Client that is configured for the specified User actor.
func (service *ActivityStream) UserClient(actorID primitive.ObjectID) streams.Client {
	return service.Client(model.ActorTypeUser, actorID)
}

// SearchDomainClient returns a streams.Client that is configured for the specified SearchDomain actor.
func (service *ActivityStream) SearchDomainClient() streams.Client {
	return service.Client(model.ActorTypeSearchDomain, primitive.NilObjectID)
}

// SearchQueryClient returns a streams.Client that is configured for the specified SearchQuery actor.
func (service *ActivityStream) SearchQueryClient(searchQueryID primitive.ObjectID) streams.Client {
	return service.Client(model.ActorTypeSearchQuery, searchQueryID)
}

// StreamClient returns a streams.Client that is configured for the specified Stream actor.
func (service *ActivityStream) StreamClient(streamID primitive.ObjectID) streams.Client {
	return service.Client(model.ActorTypeStream, streamID)
}

// AllowPrivateIPs reports whether this instance may connect to non-public
// (private/loopback) addresses. It is TRUE when the instance is served from a
// local/private hostname, so that a dev instance can federate with itself. Both
// the document-loading client stack and outbound delivery consult this single
// predicate, so loading and delivery always agree.
func (service *ActivityStream) AllowPrivateIPs() bool {
	return uri.IsLocalHostname(service.hostname)
}

// Client creates a new streams.Client that is configured for the specified actor type and ID.
func (service *ActivityStream) Client(actorType string, actorID primitive.ObjectID) streams.Client {

	userAgent := service.hostname + " /Emissary@v" + service.version + " (https://emissary.social)"

	// Build a new client stack

	// TODO: (oembed/TODO.md Phases 11.3 + RSS-FOLLOWING-RESTORE.md) When URL lookups
	// return, do NOT restore this legacy path — use sherlock's new metadata package
	// (sherlock.Client.Metadata → metadata.Card), which merges oEmbed, Open Graph,
	// Twitter Cards, and HTML signals with SSRF/body-cap guards built in.
	/* Removing legacy Sherlock lookups (RSS, oEmbed, OGP, etc) since these are not being used.
	sherlockClient := sherlock.NewClient(
		sherlock.WithKeyPairFunc(service.KeyPairFunc(actorType, actorID)),
		sherlock.WithUserAgent(userAgent),
	) */

	// If the service is on a local/private network then allow
	// the ActivityPub client to load documents from private IP addresses.
	allowPrivateIPs := service.AllowPrivateIPs()

	// Try ActivityPub documents directly
	activityPubClient := activitypub.New(
		// activitypub.WithInnerClient(sherlockClient), // Restore this to restore legacy Sherlock lookups.
		activitypub.WithKeyPairFunc(service.KeyPairFunc(actorType, actorID)),
		activitypub.WithUserAgent(userAgent),
		activitypub.WithAllowPrivateIPs(allowPrivateIPs),
	)

	// Replace 410 Gone errors with "Tombstone" documents
	tombstoneClient := tombstone.New(activityPubClient)

	// Look up WebFinger URIs. The WebFinger lookup (digit.Lookup) is a SEPARATE HTTP call from the
	// ActivityPub fetch above, so it needs the private-IP allowance threaded in independently -- without
	// it, resolving a handle for a local/private account (e.g. @user@127.0.0.1) is refused by the
	// remote transport's SSRF guard even when allowPrivateIPs is on. remote already supports this via
	// its Option API; the bug was that it was never passed here.
	allowPrivateIPsOption := remote.Option{
		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.AllowPrivateIPs(allowPrivateIPs)
			return nil
		},
	}
	webfingerClient := webfinger.New(tombstoneClient, allowPrivateIPsOption)

	// Look up BlueSky names
	bridgyfedClient := bridgyfed.New(webfingerClient)

	// Look up #Hashtags
	tagspubClient := tagspub.New(bridgyfedClient)

	// RULE: strip reserved "emissary:" properties at the trust boundary, so every layer above --
	// including the cache at rest -- only ever sees server-generated values in this namespace
	sanitizerClient := assanitizer.New(tagspubClient, model.NamespaceEmissary)

	// Enforce opinionated data formats
	normalizerClient := asnormalizer.New(sanitizerClient)

	// Apply custom caching rules to documents
	cacheRulesClient := ascacherules.New(normalizerClient)

	// Cache data in UWU DB
	cacheClient := ascache.New(
		cacheRulesClient,
		service.queue,
		service.getCommonDatabase(),
		actorType,
		actorID,
		service.hostname,
		ascache.WithIgnoreHeaders(),
	)

	// Evaluate the viewer's Rules on every result. This sits ABOVE the cache so that cache hits and
	// network fetches alike are stamped with a per-viewer verdict (hide + labels) that never touches
	// the shared cache. A document the viewer's rules hide is refused before descending (R19);
	// asrules.WithReveal is the render layer's click-to-reveal override (D2).
	rulesClient := asrules.New(cacheClient, service.ruleChecker(actorType, actorID))

	// Find inter-page IDs (like https://yo.mama.social/@sofat#main-key)
	hashClient := ashash.New(rulesClient)

	return hashClient
}

// ruleChecker returns an asrules.Checker for the given actor: it evaluates a URL -- and, once it
// loads, its document -- against the actor's Rules (the User's own plus admin rules for a User
// actor; admin-tier alone for the others, since a Stream/SearchQuery id is not a UserID). It opens
// its own session per call, because the client stack outlives any single request.
func (service *ActivityStream) ruleChecker(actorType string, actorID primitive.ObjectID) asrules.Checker {

	userID := ruleUserID(actorType, actorID)

	return func(uri string, document streams.Document) (metadata.LabelSet, error) {

		session, cancel, err := service.newSession(30 * time.Second)

		if err != nil {
			return nil, err
		}

		defer cancel()

		// Before the fetch, only the URL is known: ACTOR keys catch a blocked actor's own URL,
		// DOMAIN keys catch every URL on a blocked host. Once loaded, the document contributes its
		// own keys (author, tags) -- which is what lets a MUTE or LABEL on an author reach every
		// reply and quote fetched through this stack.
		keys := model.ActorMatchKeys(uri)

		if document.NotNil() {
			keys = append(keys, model.DocumentMatchKeys(document)...)
		}

		disposition, err := service.ruleService.DispositionForKeys(session, userID, keys, time.Now().Unix())

		if err != nil {
			return nil, err
		}

		// Translate the winning Rule(s) into the viewer's per-document label set.
		return disposition.LabelSet(), nil
	}
}

// ruleUserID maps a client's actor to the UserID its fetches are gated by: the User's own id for a
// User actor, or NilObjectID (admin-tier rules only) for the Application/Stream/Search actors, whose
// ids are not UserIDs.
func ruleUserID(actorType string, actorID primitive.ObjectID) primitive.ObjectID {

	if actorType == model.ActorTypeUser {
		return actorID
	}

	return primitive.NilObjectID
}

/******************************************
 * Hannibal HTTP Client Interface
 ******************************************/

// Save adds a single document to the ActivityStream cache
func (service *ActivityStream) Save(document streams.Document) error {
	return service.AppClient().Save(document)
}

// Delete removes a single document from the database by its URL
func (service *ActivityStream) Delete(url string) error {
	return service.AppClient().Delete(url)
}

/******************************************
 * Custom Query Methods
 ******************************************/

// Range iterates over the cached ActivityStream documents that match the provided criteria
func (service *ActivityStream) Range(ctx context.Context, criteria exp.Expression, options ...option.Option) iter.Seq[ascache.Value] {

	const location = "service.ActivityStream.Range"

	return func(yield func(ascache.Value) bool) {

		// Connect to the database
		collection, err := service.collection(ctx)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Connecting to database"))
			return
		}

		// Query the database
		iterator, err := collection.Iterator(criteria, options...)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Querying database", criteria))
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

// QueryByContext returns a slice of streams.Document values that are associated with the specified context name.
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

// QueryActors returns a slice of ActorSummary values that match the provided query string.
func (service *ActivityStream) QueryActors(queryString string) ([]model.ActorSummary, error) {

	const location = "service.ActivityStream.QueryActors"

	// If we think this is a URI  we can use then try to retrieve it directly.
	if service.looksLikeValidURI(queryString) {

		// Try to load the actor directly from the Interwebs
		if object, err := service.AppClient().Load(queryString, sherlock.AsActor()); err == nil {

			if object.IsActor() {

				// If this is a valid, but (previously) unknown actor, then add it to the results
				// This will also automatically get cached/crawled for next time.
				result := []model.ActorSummary{{
					ID:                object.ID(),
					Type:              object.Type(),
					Name:              object.Name(),
					Icon:              object.Icon().Href(),
					PreferredUsername: object.PreferredUsername(),
				}}

				return result, nil
			}
		}
	}

	// Fall through means that we can't find a perfect match, so fall back to a full-text search
	ctx, cancel := timeoutContext(2)
	defer cancel()

	collection, err := service.collection(ctx)

	if err != nil {
		return nil, derp.Wrap(err, location, "Connecting to database")
	}

	// Get [top 6] matching actors from the database
	result, err := queries.SearchActivityStreamActors(collection, queryString)

	if err != nil {
		return nil, derp.Wrap(err, location, "Querying database")
	}

	// Done? Done.
	return result, nil
}

func (service *ActivityStream) looksLikeValidURI(uri string) bool {

	if sherlock.IsValidAddress(uri) {
		return true
	}

	if bridgyfed.LooksLikeBluesky(uri) {
		return true
	}

	if isValid, _ := tagspub.IsHashtag(uri); isValid {
		return true
	}

	return false
}

// QueryReplies returns a slice of streams.Document values that are replies to the specified document, and were published before the specified date.
func (service *ActivityStream) QueryReplies(ctx context.Context, inReplyTo string, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation(ctx, "Reply", inReplyTo, "after", 0, done)
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

// QueryLikesBeforeDate returns a channel of "Like" documents that target the specified URL, published before the specified date.
func (service *ActivityStream) QueryLikesBeforeDate(ctx context.Context, relationHref string, maxDate int64, done <-chan struct{}) <-chan streams.Document {
	return service.queryByRelation(ctx, vocab.ActivityTypeLike, relationHref, "before", maxDate, done)
}

// queryByRelation returns a channel of streams.Document values related to the specified URL,
// cut off "before" or "after" the specified date.
func (service *ActivityStream) queryByRelation(ctx context.Context, relationType string, relationHref string, cutType string, cutDate int64, done <-chan struct{}) <-chan streams.Document {

	const location = "service.ActivityStream.QueryRelated"
	const publishedField = "object.published"

	result := make(chan streams.Document)

	go func() {

		defer close(result)

		// Build the query
		criteria := exp.
			Equal("metadata.relationType", relationType).
			AndEqual("metadata.relationHref", relationHref)

		var sortOption option.Option

		if cutType == "before" {
			criteria = criteria.AndLessThan(publishedField, time.Unix(cutDate, 0))
			sortOption = option.SortDesc(publishedField)
		} else {
			criteria = criteria.AndGreaterThan(publishedField, time.Unix(cutDate, 0))
			sortOption = option.SortAsc(publishedField)
		}

		// Try to query the database
		documents, err := service.documentIterator(ctx, criteria, sortOption)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Querying database"))
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
					streams.WithClient(service.AppClient()),
				)
			}
		}
	}()

	return result
}

// GetActor retrieves an actor from their handle or URL
func (service *ActivityStream) GetActor(actor string) (streams.Document, error) {

	const location = "service.ActivityStream.GetActor"

	// Try to load the actor as a JSON-LD document
	document, err := service.AppClient().Load(actor, sherlock.AsActor())

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Loading ActivityPub Actor", actor)
	}

	// RULE: Verify that this document is an Actor (not a document or activity)
	if !document.IsActor() {
		return streams.NilDocument(), derp.NotFound(location, "Recipient is not an ActivityPub Actor", actor)
	}

	return document, nil
}

// GetRecipient retrieves the recipient's ID and inbox URL
func (service *ActivityStream) GetRecipient(recipient string) (string, string, error) {

	const location = "service.ActivityStream.GetRecipient"

	document, err := service.GetActor(recipient)

	if err != nil {
		return "", "", derp.Wrap(err, location, "Loading ActivityPub Actor", recipient)
	}

	// Successssssssss.
	return document.ID(), document.Inbox().String(), nil
}

// Signature verification lives in activityStream_signature.go, which owns both the key lookup and
// the rules about when a key may be re-fetched.

// KeyPairFunc returns a function that will locate the public/private key pair
// for the specidied URL.  This can only be used for local URLs
func (service *ActivityStream) KeyPairFunc(actorType string, actorID primitive.ObjectID) func() (string, crypto.PrivateKey) {

	const location = "service.ActivityStream.KeyPairFunc"

	return func() (string, crypto.PrivateKey) {

		session, cancel, err := service.newSession(10 * time.Second)
		defer cancel()

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Connecting to database"))
			return "", nil
		}

		// USE service.actorType and service.actorID to retrieve the required PEM keys.
		publicKeyID, privateKey, err := service.locatorService.GetPrivateKey(session, actorType, actorID)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Retrieving private key"))
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
		return nil, derp.Wrap(err, location, "Querying database", criteria, derp.WithInternalError())
	}

	if collection == nil {
		return nil, derp.Internal(location, "Collection cannot be nil. This should never happen.")
	}

	return collection.Iterator(criteria, options...)
}

// collection creates a new mongodb Session and returns the mongodb Collection that stores ActivityStream documents
func (service *ActivityStream) collection(ctx context.Context) (data.Collection, error) {

	const location = "service.ActivityStream.collection"

	// NILCHECK: the common-database getter is only nil before Refresh has run
	if service.getCommonDatabase == nil {
		return nil, derp.Internal(location, "Service not initialized")
	}

	// Connect to the database (read live through the getter -- see the field comment)
	session, err := service.getCommonDatabase().Session(ctx)

	if err != nil {
		return nil, derp.Wrap(err, location, "Connecting to database", derp.WithInternalError())
	}

	// NILCHECK: session cannot be nil.
	if session == nil {
		return nil, derp.Internal(location, "Database session is nil. This should never happen.")
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
