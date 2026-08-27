package service

import (
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// maxReplyDepth bounds how far the provenance walk climbs an inReplyTo chain. A malicious cycle or an
// absurdly deep thread would otherwise recurse until the stack overflows -- a remote-triggerable DoS.
const maxReplyDepth = 32

// SaveNewsItem adds/updates a NewsItem for a followed document, after walking its provenance chain to
// the primary post and dropping anything authored (or delivered) by a blocked or muted identity, or
// carrying a blocked or muted hashtag (R18, D12).
func (service *Following) SaveNewsItem(session data.Session, following *model.Following, document streams.Document, originType string) error {

	const location = "service.Following.SaveNewsItem"

	// Walk `Create`/`Update`/`inReplyTo` back to the primary document, filtering the whole provenance
	// chain against this User's rules along the way.
	walk := &primaryPostWalk{
		ruleService: service.ruleService,
		session:     session,
		userID:      following.UserID,
		now:         time.Now().Unix(),
		seen:        make(map[string]bool),
	}

	original, originType, dropped, err := walk.primaryPost(document, originType, 0)

	if err != nil {
		return derp.Wrap(err, location, "Walking provenance chain", document.ID())
	}

	// `dropped` means a blocked/muted identity or hashtag was found anywhere in the chain; a nil
	// `original` means the chain terminated in a `Delete`/`Undo`. Either way there are no newsfeed
	// side effects (no item, no reply-thread SSE).
	if dropped || original.IsNil() {
		return nil
	}

	// Send SSE notifications to any local stream referenced in the document's "inReplyTo" -- behind the
	// rule gate, so a blocked/muted reply never nudges a thread view.
	service.streamService.NotifyInReplyTo(session, document.InReplyTo().ID())

	// Convert the document into a newsItem (and traverse responses if necessary)
	newsItem := getNewsItem(following.UserID, original)
	newsItem.Context = original.Context()
	newsItem.FollowingID = following.FollowingID
	newsItem.FolderID = following.FolderID.Value()
	newsItem.AddReference(following.Origin(originType))

	// Try to save a unique version of this newsItem to the database (always collapse duplicates)
	if err := service.saveUniqueNewsItem(session, newsItem); err != nil {
		return derp.Wrap(err, location, "Saving newsItem", newsItem)
	}

	// Crawl the document's context/reply chain in the background (post-commit).  The
	// signature collapses duplicate crawls of the same document -- the same reply
	// arriving via many local followers must seed ONE crawl, not one per follower.
	postcommit.Publish(
		session,
		service.queue,
		"CrawlContext",
		mapof.Any{"url": document.ID(), "hostname": service.hostname},
		queue.WithSignature("CrawlContext:"+service.hostname+":"+document.ID()),
	)

	// Yee. Haw.
	return nil
}

// saveUnique adds/updates a message in the database.  If the message.URL does not already
// exist, then a new message is added to the Inbox.  Otherwise, the "references" data will
// of the existing record be updated and the unique value will be re-saved.
func (service *Following) saveUniqueNewsItem(session data.Session, message model.NewsItem) error {

	const location = "service.Following.saveUnique"

	// Search for a previous UNREAD message with our same UserID and URL.
	previousNewsItem := model.NewsItem{}

	if err := service.newsFeedService.LoadByURL(session, message.UserID, message.URL, &previousNewsItem); err != nil {
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Searching for duplicate message", message)
		}
	}

	// If no previous message was found, then save the current message as is
	if previousNewsItem.IsNew() {

		if err := service.newsFeedService.Save(session, &message, "Created"); err != nil {
			return derp.Wrap(err, location, "Saving new message", message)
		}

		return nil
	}

	// Fall through means that we have a duplicate message.

	// Try to update the previousNewsItem with a new origin (a new reply, like, etc)
	isReferenceUpdated := previousNewsItem.AddReference(message.Origin) // nolint:scopeguard (readability)
	isStatusUpdated := false

	// Update the message status to "NEW-REPLIES" so that previously
	// read messages will show up again in the Inbox.
	if message.Origin.Type == model.OriginTypeReply {
		isStatusUpdated = previousNewsItem.MarkNewReplies()
	}

	// if the message was updated (from AddReference or MarkNewReplies) then save it.
	if isReferenceUpdated || isStatusUpdated {
		if err := service.newsFeedService.Save(session, &previousNewsItem, "NewsItem Imported"); err != nil {
			return derp.Wrap(err, location, "Updating previous message with new origin and status", previousNewsItem)
		}
	}

	// Successfully updated the message, or not.  But still, it's good.
	return nil
}

/******************************************
 * Provenance Walk
 ******************************************/

// primaryPostWalk carries the state of a single provenance traversal: the rule check it applies at
// every identity, the User it runs for, and the cycle/depth guards that bound the recursion.
type primaryPostWalk struct {
	ruleService *Rule
	session     data.Session
	userID      primitive.ObjectID
	now         int64
	seen        map[string]bool
}

// primaryPost traverses UP a chain of Activities and replies to the first message that was posted. It
// returns that document, the accumulated originType (REPLY if any hop was a reply), and a `dropped`
// flag. `dropped` is TRUE when any identity in the chain -- a booster, an author, or the host of a
// link about to be fetched -- is blocked or muted (R18), or when any document in the chain carries a
// blocked or muted hashtag (D12); the caller must then create no newsfeed item. A nil document with
// dropped=FALSE means the chain simply terminated in a subtractive activity.
func (w *primaryPostWalk) primaryPost(document streams.Document, originType string, depth int) (streams.Document, string, bool, error) {

	// RULE: bound the recursion so a hostile inReplyTo cycle cannot overflow the stack.
	if depth >= maxReplyDepth {
		return streams.NilDocument(), "", false, nil
	}

	// Activities unwrap to their object; subtractive activities produce no newsfeed item.
	switch document.Type() {

	case vocab.ActivityTypeAdd, vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate:
		return w.primaryPost(document.Object().LoadLink(), originType, depth+1)

	case vocab.ActivityTypeAnnounce:
		// RULE: an Announce carries its OWN actor; a blocked/muted booster's boost creates nothing (R2).
		// Checked before unwrapping, because the announcer's identity is discarded past this point.
		filtered, err := w.actorFiltered(document.ActorID())

		if err != nil {
			return streams.NilDocument(), "", false, err
		}

		if filtered {
			return streams.NilDocument(), "", true, nil
		}

		return w.primaryPost(document.Object().LoadLink(), model.OriginTypeAnnounce, depth+1)

	case vocab.ActivityTypeDislike:
		return w.primaryPost(document.Object().LoadLink(), model.OriginTypeDislike, depth+1)

	case vocab.ActivityTypeLike:
		return w.primaryPost(document.Object().LoadLink(), model.OriginTypeLike, depth+1)

	case vocab.ActivityTypeDelete, vocab.ActivityTypeRemove, vocab.ActivityTypeUndo:
		return streams.NilDocument(), "", false, nil
	}

	// Fall through: this is an Object (Note/Article), not an Activity.

	// RULE: drop the item if this Object is filtered -- its author is blocked/muted, it carries a
	// blocked/muted hashtag, or it quotes blocked/muted content (R18, D12).
	filtered, err := w.objectFiltered(document)

	if err != nil {
		return streams.NilDocument(), "", false, err
	}

	if filtered {
		return streams.NilDocument(), "", true, nil
	}

	// Cycle guard: stop climbing (but keep this document) if we have already visited it.
	if id := document.ID(); id != "" {
		if w.seen[id] {
			return document, originType, false, nil
		}
		w.seen[id] = true
	}

	// Walk UP the reply chain to the primary post.
	return w.climbReplyChain(document, originType, depth)
}

// climbReplyChain resolves the parent of a reply, returning the primary post found upthread or the
// document itself. It propagates a `dropped` verdict from any blocked/muted ancestor -- distinct from a
// nil-with-dropped=false result, which just means no primary was found (subtractive or depth-limited).
func (w *primaryPostWalk) climbReplyChain(document streams.Document, originType string, depth int) (streams.Document, string, bool, error) {

	inReplyTo := document.InReplyTo()

	if inReplyTo.IsNil() {
		return document, originType, false, nil
	}

	// Change origin type from PRIMARY to REPLY without affecting LIKE and DISLIKE types.
	if originType == model.OriginTypePrimary {
		originType = model.OriginTypeReply
	}

	// RULE: pre-fetch host check (privacy, not optimization). The next LoadLink is signed with the
	// User's own key, so dereferencing a blocked host would tell a blocked server who is reading.
	hostFiltered, err := w.hostFiltered(inReplyTo.ID())

	if err != nil {
		return streams.NilDocument(), "", false, err
	}

	if hostFiltered {
		return streams.NilDocument(), "", true, nil
	}

	// Traverse up the tree.
	parent, parentOrigin, dropped, err := w.primaryPost(inReplyTo.LoadLink(), originType, depth+1)

	if err != nil {
		return streams.NilDocument(), "", false, err
	}

	// A blocked/muted ancestor drops the whole item -- propagate that verdict up.
	if dropped {
		return streams.NilDocument(), "", true, nil
	}

	// A primary found upthread wins; otherwise this document is itself the primary.
	if parent.NotNil() {
		return parent, parentOrigin, false, nil
	}

	return document, originType, false, nil
}

// objectFiltered returns TRUE if an Object node should drop the item: its own disposition (author
// identity or content hashtags) is blocked/muted, or it quotes blocked/muted content (R18, D12).
func (w *primaryPostWalk) objectFiltered(document streams.Document) (bool, error) {

	filtered, err := w.documentFiltered(document)

	if err != nil {
		return false, err
	}

	if filtered {
		return true, nil
	}

	return w.quoteFiltered(document)
}

// quoteFiltered returns TRUE if any post this document quotes is blocked or muted (R18) -- otherwise a
// blocked author reaches the feed as the quoted body of an allowed post. Quotes ride non-standard
// fields, so they are extracted explicitly. Each quote is checked by host first (no fetch), then by
// resolving the quoted object and checking its full disposition (author and hashtags); an
// unresolvable quote fails open (a broken quote must not drop an allowed post).
func (w *primaryPostWalk) quoteFiltered(document streams.Document) (bool, error) {

	for _, url := range quoteURLs(document) {

		// Pre-fetch host check: a quote of a blocked DOMAIN drops the item without a fetch.
		hostFiltered, err := w.hostFiltered(url)

		if err != nil {
			return false, err
		}

		if hostFiltered {
			return true, nil
		}

		// Resolve the quoted object (via this document's cache-and-block-aware client) and check its
		// full disposition. A fetch failure -- network error, or asrules refusing a blocked origin --
		// fails open.
		quoted, err := document.Client().Load(url)

		if err != nil {
			continue
		}

		quotedFiltered, err := w.documentFiltered(quoted)

		if err != nil {
			return false, err
		}

		if quotedFiltered {
			return true, nil
		}
	}

	return false, nil
}

// quoteURLs returns the URLs a post quotes, gathered from the non-standard fields that carry
// quote-posts across vocabularies: the Misskey/Fedibird `quoteUrl`/`quoteUri`/`_misskey_quote` string
// fields, and FEP-e232 `Link` tags whose `rel` names a quote (the target being the tag's `href`).
func quoteURLs(document streams.Document) []string {

	result := make([]string, 0)

	for _, property := range []string{"quoteUrl", "quoteUri", "_misskey_quote"} {
		if url := document.Get(property).String(); url != "" {
			result = append(result, url)
		}
	}

	for tag := document.Tag(); tag.NotNil(); tag = tag.Next() {

		if tag.IsString() || (tag.Type() != vocab.CoreTypeLink) {
			continue
		}

		if href := tag.Href(); (href != "") && strings.Contains(tag.Rel().String(), "quote") {
			result = append(result, href)
		}
	}

	return result
}

// actorFiltered returns TRUE if the given actor URI is blocked or muted for this walk's User. An empty
// URI matches nothing.
func (w *primaryPostWalk) actorFiltered(actorID string) (bool, error) {

	if actorID == "" {
		return false, nil
	}

	disposition, err := w.ruleService.DispositionForKeys(w.session, w.userID, model.ActorMatchKeys(actorID), w.now)

	if err != nil {
		return false, err
	}

	return disposition.IsFiltered(), nil
}

// documentFiltered returns TRUE if this document's own disposition is blocked or muted: ONE indexed
// rules query over DocumentMatchKeys, which names the document's author (`attributedTo` and `actor`)
// AND its content (Hashtag TAG keys, D12). Reads only loaded fields -- the document is already
// resolved by the time the walk reaches this check.
func (w *primaryPostWalk) documentFiltered(document streams.Document) (bool, error) {

	disposition, err := w.ruleService.Disposition(w.session, w.userID, document, w.now)

	if err != nil {
		return false, err
	}

	return disposition.IsFiltered(), nil
}

// hostFiltered returns TRUE if the host of the given URL is domain-blocked or domain-muted. It checks
// DOMAIN keys only, because pre-fetch that host is the only identity known (the author is not yet loaded).
func (w *primaryPostWalk) hostFiltered(url string) (bool, error) {

	disposition, err := w.ruleService.DispositionForKeys(w.session, w.userID, model.DomainMatchKeys(url), w.now)

	if err != nil {
		return false, err
	}

	return disposition.IsFiltered(), nil
}

/******************************************
 * Helper Functions
 ******************************************/

// getNewsItem returns an inbox NewsItem object based on the provided arguments.
func getNewsItem(userID primitive.ObjectID, document streams.Document) model.NewsItem {

	result := model.NewNewsItem()
	result.UserID = userID
	result.Context = document.Context()
	result.SocialRole = document.Type()
	result.URL = document.ID()
	result.InReplyTo = document.InReplyTo().ID()
	result.PublishDate = document.Published().Unix()

	return result
}
