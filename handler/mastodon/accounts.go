package mastodon

import (
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// loadUserByAccountID loads a local User by a Mastodon account ID, which may be either
// form in use: the short local hex UserID (what model.User.Toot() returns) or a full
// actor URL (what a client may still hold for a remote account, or an older cached
// value). See the comment on model.User.Toot() for why local accounts use a short ID.
func loadUserByAccountID(factory *service.Factory, session data.Session, id string) (model.User, error) {

	user := model.NewUser()

	if objectID, err := primitive.ObjectIDFromHex(id); err == nil {
		if err := factory.User().LoadByID(session, objectID, &user); err == nil {
			return user, nil
		}
	}

	err := factory.User().LoadByProfileURL(session, id, &user)
	return user, err
}

// resolveAccountURL resolves a Mastodon account ID to the account's real,
// federatable profile URL. It accepts the ID forms Emissary hands out or might
// receive:
//
//   - a local User's hex ID          -> that User's ActivityPub URL
//   - an encoded remote ID ("u_...")  -> the actor URL packed into it (no DB lookup)
//   - a bare actor URL                -> itself (e.g. a status's embedded author)
//
// Anything else is an unknown ID and returns derp.NotFound. Used wherever a real
// URL is required (e.g. creating a Following record), not just for loading a
// local User row.
func resolveAccountURL(factory *service.Factory, session data.Session, id string) (string, error) {

	const location = "handler.mastodon_resolveAccountURL"

	// Local account?
	if user, err := loadUserByAccountID(factory, session, id); err == nil {
		return user.ActivityPubURL(), nil
	}

	// One of our encoded remote IDs? Decode it back to the actor URL with no
	// database lookup, so it stays valid however many times the actor's ascache
	// row has been recycled.
	if actorURL, ok := model.DecodeRemoteAccountID(id); ok {
		return actorURL, nil
	}

	// A bare actor URL (e.g. the embedded author of a timeline status, before
	// PersonLink.Toot() is switched to the encoded form).
	if parsed, err := url.Parse(id); err == nil && parsed.IsAbs() {
		return id, nil
	}

	return "", derp.NotFound(location, "Unrecognized account ID", id)
}

// resolveAccountID returns a stable Mastodon account ID for an actor URL: the
// local User's own hex ID for a local account (matching model.User.Toot()), or
// the URL encoded into an opaque "u_..." token for a remote one. The remote form
// is a pure function of the URL -- NOT the ascache row's _id, which is reminted
// on every refetch -- so an ID handed to a client never goes stale.
func resolveAccountID(factory *service.Factory, session data.Session, actorURL string) string {

	if user, err := loadUserByAccountID(factory, session, actorURL); err == nil {
		return user.UserID.Hex()
	}

	return model.EncodeRemoteAccountID(actorURL)
}

// mapDocumentToAccount maps a fetched remote actor document to a Mastodon Account.
// Shared by GetAccount (looks up by opaque ID -- no handle available) and
// GetAccount_Lookup (looks up by handle -- caller already knows the exact "acct" to
// use, so it overwrites this function's derived value afterward; see that function).
func mapDocumentToAccount(factory *service.Factory, session data.Session, document streams.Document) object.Account {

	// Acct has no client-supplied value here, so derive "username@domain" from the
	// document itself, matching the Mastodon spec's format for a remote account.
	acct := document.PreferredUsername()

	if parsed, err := url.Parse(document.URL()); err == nil && parsed.Hostname() != "" {
		acct += "@" + parsed.Hostname()
	}

	// The real Account entity requires a non-null created_at (confirmed against a
	// real client's Codable model -- see the same rule applied in PersonLink.Toot()).
	// Use the actor's own "published" date when the document has one; otherwise fall
	// back to now rather than crash or lie with a zero/epoch date.
	createdAt := document.Published()

	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	avatar := document.Icon().URL()
	header := document.Image().URL()

	// Best-effort follower/following/post counts: the actor document only carries
	// the collection URLs, so fetch each and read its "totalItems". A failure (a
	// server that hides these, a 404) just leaves the count at 0, which is what
	// Mastodon itself does. ascache serves repeat profile views without refetching.
	return object.Account{
		ID:             resolveAccountID(factory, session, document.ID()),
		Acct:           acct,
		Username:       document.PreferredUsername(),
		DisplayName:    document.Name(),
		Avatar:         avatar,
		AvatarStatic:   avatar,
		Header:         header,
		HeaderStatic:   header,
		URL:            document.URL(),
		Note:           document.Summary(),
		CreatedAt:      model.MastodonDate(createdAt),
		FollowersCount: document.Followers().LoadLink().TotalItems(),
		FollowingCount: document.Following().LoadLink().TotalItems(),
		StatusesCount:  document.Outbox().LoadLink().TotalItems(),
	}
}

/*******************************************
 * Mastodon API - Account Handlers
 * https://docs.joinmastodon.org/methods/accounts/
 *******************************************/

// PostAccount implements the Mastodon "register an account" endpoint
func PostAccount(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount) (object.Token, error) {

	const location = "handler.mastodon_PostAccount"

	return func(auth model.Authorization, t txn.PostAccount) (object.Token, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Token{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Token{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Confirm that the domain is accepting new users
		if !factory.Domain().HasRegistrationForm() {
			return object.Token{}, derp.Forbidden(location, "Signup is not allowed on this domain")
		}

		if !t.Agreement {
			return object.Token{}, derp.Forbidden(location, "You must agree to the terms of service")
		}

		// Create a new User account
		userService := factory.User()
		user := model.NewUser()
		user.Username = t.Username
		user.EmailAddress = t.Email
		user.Locale = t.Locale
		user.SignupNote = t.Reason

		if err := factory.Steranko(session).SetPassword(&user, t.Password); err != nil {
			return object.Token{}, derp.Wrap(err, location, "Setting password")
		}

		if err := userService.Save(session, &user, "Created via Mastodon API"); err != nil {
			return object.Token{}, derp.Wrap(err, location, "Saving user")
		}

		// Create a new OAuth token
		oauthUserTokenService := factory.OAuthUserToken()
		token, err := oauthUserTokenService.CreateFromUser(session, &user, auth.ClientID, auth.Scope)

		if err != nil {
			return object.Token{}, derp.Wrap(err, location, "Creating OAuth token")
		}

		return token.Toot(), nil
	}
}

// GetAccount_VerifyCredentials implements the Mastodon "verify account credentials" endpoint
func GetAccount_VerifyCredentials(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_VerifyCredentials) (object.Account, error) {

	const location = "handler.mastodon_GetAccount_VerifyCredentials"

	return func(auth model.Authorization, t txn.GetAccount_VerifyCredentials) (object.Account, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(session, auth.UserID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized User")
		}

		// Return as a Toot
		return user.Toot(), nil
	}
}

// PatchAccount_UpdateCredentials implements the Mastodon "update account credentials" endpoint
func PatchAccount_UpdateCredentials(serverFactory *server.Factory) func(model.Authorization, txn.PatchAccount_UpdateCredentials) (object.Account, error) {

	const location = "handler.mastodon_PatchAccount_UpdateCredentials"

	return func(auth model.Authorization, t txn.PatchAccount_UpdateCredentials) (object.Account, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(session, auth.UserID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized User")
		}

		// Update the User's information
		user.DisplayName = t.DisplayName
		user.Note = t.Note
		user.IsPublic = t.Discoverable

		if err := userService.Save(session, &user, "Updated via Mastodon API"); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Saving user")
		}

		// Return updated JSON
		return user.Toot(), nil
	}
}

// GetAccounts implements the Mastodon batch "get multiple accounts" endpoint
// (GET /api/v1/accounts?id[]=...). The client uses it to refresh accounts it
// already has on screen -- opening a profile reached from a post author or a
// list -- so it must return the same fully-populated Account that GetAccount
// does. An id that no longer resolves is skipped rather than failing the batch.
func GetAccounts(serverFactory *server.Factory) func(model.Authorization, txn.GetAccounts) ([]object.Account, error) {

	const location = "handler.mastodon_GetAccounts"

	return func(auth model.Authorization, t txn.GetAccounts) ([]object.Account, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		result := make([]object.Account, 0, len(t.IDs))

		for _, id := range t.IDs {

			// A local account
			if user, err := loadUserByAccountID(factory, session, id); err == nil {
				result = append(result, user.Toot())
				continue
			}

			// A remote account: resolve + dereference, same as GetAccount.
			accountURL, err := resolveAccountURL(factory, session, id)

			if err != nil {
				continue
			}

			document, err := factory.ActivityStream().UserClient(auth.UserID).Load(accountURL)

			if err != nil {
				continue
			}

			result = append(result, mapDocumentToAccount(factory, session, document))
		}

		return result, nil
	}
}

// GetAccount implements the Mastodon "get account" endpoint
func GetAccount(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount) (object.Account, error) {

	const location = "handler.mastodon_GetAccount"

	return func(auth model.Authorization, t txn.GetAccount) (object.Account, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the User -- a local account, if there's a User row for this ID.
		if user, err := loadUserByAccountID(factory, session, t.ID); err == nil {
			return user.Toot(), nil
		}

		// Not local -- try resolving to a cached remote actor's real URL. This also
		// covers a stale/unresolvable ID (see resolveAccountURL), which fails here
		// with a proper error instead of being treated as a URL.
		accountURL, err := resolveAccountURL(factory, session, t.ID)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized account", t.ID)
		}

		client := factory.ActivityStream().UserClient(auth.UserID)
		document, err := client.Load(accountURL)

		if err != nil {
			// RULE: derp.Wrap inherits the wrapped error's status code by default, and
			// a remote origin's own failure (401, 403, 429...) is not our caller's
			// fault. Passing it through as-is would make the client think its OWN
			// bearer token is invalid. Report it as what it actually is: we could
			// not reach the remote account.
			return object.Account{}, derp.Wrap(err, location, "Loading remote account", accountURL, derp.WithBadGateway())
		}

		return mapDocumentToAccount(factory, session, document), nil
	}
}

// GetAccount_Statuses implements the Mastodon "get account statuses" endpoint
func GetAccount_Statuses(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Statuses) ([]object.Status, toot.PageInfo, error) {

	const location = "handler.mastodon_GetAccount_Statuses"

	return func(auth model.Authorization, t txn.GetAccount_Statuses) ([]object.Status, toot.PageInfo, error) {

		// Emissary has no featured (pinned) posts
		if t.Pinned {
			return []object.Status{}, toot.PageInfo{}, nil
		}

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the requested User
		user, err := loadUserByAccountID(factory, session, t.ID)

		if err != nil {

			// Not a local account. If it still resolves to something real (a cached
			// remote actor), the ID is valid -- serve what the News Feed holds.
			if accountURL, resolveErr := resolveAccountURL(factory, session, t.ID); resolveErr == nil {
				return remoteAccountStatuses(factory, session, auth, t, accountURL)
			}

			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Unrecognized User")
		}

		// Query all posts by this user that are visible to the caller
		streamService := factory.Stream()
		streams, err := streamService.QueryByUser(session, auth, user.UserID, queryExpression(t), option.MaxRows(pageLimit(t.Limit)))

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Querying streams")
		}

		// QueryByUser filters and sorts on createDate, so the paging cursors must be
		// createDate too -- Stream.GetRank() is its "rank", which getPageInfo would use.
		pageInfo := toot.PageInfo{}

		if length := len(streams); length > 0 {
			pageInfo.MaxID = strconv.FormatInt(streams[length-1].CreateDate, 10)
			pageInfo.MinID = strconv.FormatInt(streams[0].CreateDate, 10)
		}

		// Return posts as toot.Status(es)
		return getSliceOfToots(streams), pageInfo, nil
	}
}

// remoteAccountStatuses returns a remote account's recent public posts, read live
// from its ActivityPub outbox. It falls back to whatever the News Feed holds when
// the outbox can't be read (a server that hides it, a network failure) or is empty.
//
// Posts already in the News Feed keep their NewsItem ID, so favourite/boost keep
// working on them. Any other post is identified by its encoded URL, which those
// endpoints can't resolve to a NewsItem yet.
//
// The outbox has no stable cursor to map onto max_id/min_id, so this returns one
// page, newest first, with no paging info.
func remoteAccountStatuses(factory *service.Factory, session data.Session, auth model.Authorization, t txn.GetAccount_Statuses, accountURL string) ([]object.Status, toot.PageInfo, error) {

	client := factory.ActivityStream().UserClient(auth.UserID)
	limit := int(pageLimit(t.Limit))

	actor, err := client.Load(accountURL)

	if err != nil {
		return accountStatusesFromNewsFeed(factory, session, auth, t, accountURL)
	}

	account := mapDocumentToAccount(factory, session, actor)
	outbox := actor.Outbox().LoadLink()
	newsFeedService := factory.NewsFeed()
	accounts := newAccountMemo()
	result := make([]object.Status, 0, limit)

	// Replies and boosts are skipped as we go, so scan a few pages' worth
	for item := range collections.RangeDocuments(outbox, collections.WithMaxDocuments(limit*3)) {

		if len(result) >= limit {
			break
		}

		if item.Type() != vocab.ActivityTypeCreate {
			continue
		}

		post := item.UnwrapActivity()

		if post.ID() == "" || post.Content() == "" && post.Attachment().IsNil() {
			continue
		}

		if t.ExcludeReplies && post.InReplyTo().ID() != "" {
			continue
		}

		var status object.Status

		newsItem := model.NewNewsItem()

		if err := newsFeedService.LoadByURL(session, auth.UserID, post.ID(), &newsItem); err == nil {
			status, _ = newsItemToStatus(client, factory, session, accounts, newsItem)
		} else {
			status = documentToStatus(post, account)
		}

		if t.OnlyMedia && len(status.MediaAttachments) == 0 {
			continue
		}

		result = append(result, status)
	}

	if len(result) == 0 {
		return accountStatusesFromNewsFeed(factory, session, auth, t, accountURL)
	}

	return result, toot.PageInfo{}, nil
}

// documentToStatus builds a Status straight from a post document, for posts that
// have no NewsItem. The ID is the encoded post URL (see model.EncodeRemoteStatusID).
func documentToStatus(document streams.Document, account object.Account) object.Status {

	url := document.URL()

	if url == "" {
		url = document.ID()
	}

	summary := document.Summary()

	return object.Status{
		ID:               model.EncodeRemoteStatusID(document.ID()),
		URI:              document.ID(),
		URL:              url,
		CreatedAt:        model.MastodonDate(document.Published()),
		Visibility:       "public",
		Account:          account,
		Content:          document.Content(),
		SpoilerText:      summary,
		Sensitive:        summary != "",
		MediaAttachments: mapDocumentToMediaAttachments(document),
	}
}

// accountStatusesFromNewsFeed returns the posts that a remote account has authored
// and that this User has received. Emissary keeps no remote account's post history,
// so this is limited to what has reached the User's News Feed.
//
// RULE: NewsItem.Origin is whoever led us to the post -- its author for a PRIMARY or
// REPLY, but the booster for an ANNOUNCE. Only the first two are the account's own.
func accountStatusesFromNewsFeed(factory *service.Factory, session data.Session, auth model.Authorization, t txn.GetAccount_Statuses, accountURL string) ([]object.Status, toot.PageInfo, error) {

	const location = "handler.mastodon_accountStatusesFromNewsFeed"

	originTypes := []string{model.OriginTypePrimary}

	if !t.ExcludeReplies {
		originTypes = append(originTypes, model.OriginTypeReply)
	}

	criteria := queryExpressionByField(t, "rank").
		AndEqual("origin.url", accountURL).
		And(exp.In("origin.type", originTypes))

	newsItems, err := factory.NewsFeed().QueryByUserID(session, auth.UserID, criteria, option.SortDesc("rank"), option.MaxRows(pageLimit(t.Limit)))

	if err != nil {
		return nil, toot.PageInfo{}, derp.Wrap(err, location, "Querying news feed", accountURL)
	}

	// Paging cursors come from the full page, so filtering below can't stall the client
	pageInfo := getPageInfo(newsItems)
	statuses := newsItemsToPosts(factory, session, auth, newsItems)

	if t.OnlyMedia {
		statuses = slices.DeleteFunc(statuses, func(status object.Status) bool {
			return len(status.MediaAttachments) == 0
		})
	}

	return statuses, pageInfo, nil
}

// GetAccount_Followers implements the Mastodon "get account followers" endpoint, and always returns an empty list
func GetAccount_Followers(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Followers) ([]object.Account, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetAccount_Followers) ([]object.Account, toot.PageInfo, error) {

		// Emissary does not (currently?) publish followers
		return []object.Account{}, toot.PageInfo{}, nil
	}
}

// GetAccount_Following implements the Mastodon "get account following" endpoint.
// Emissary only knows one account's following graph -- the local User's own -- so
// this returns that list when the caller asks for their own account, and an
// honest empty result for anyone else (the same cross-account limit Mastodon
// itself has).
func GetAccount_Following(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Following) ([]object.Account, toot.PageInfo, error) {

	const location = "handler.mastodon_GetAccount_Following"

	return func(auth model.Authorization, t txn.GetAccount_Following) ([]object.Account, toot.PageInfo, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Only the caller's own following list is available.
		if user, err := loadUserByAccountID(factory, session, t.ID); err != nil || user.UserID != auth.UserID {
			return []object.Account{}, toot.PageInfo{}, nil
		}

		records, err := factory.Following().RangeByUserID(session, auth.UserID)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Querying following")
		}

		result := make([]object.Account, 0)
		client := factory.ActivityStream().UserClient(auth.UserID)

		for following := range records {

			profileURL := following.ProfileURL
			if profileURL == "" {
				profileURL = following.URL
			}

			// Prefer a live (ascache-backed) dereference so the row carries real
			// follower/following/post counts, same as a direct GetAccount fetch.
			// Some origins are unreachable; fall back to the Following row's own
			// data rather than dropping the account from the list.
			if document, err := client.Load(profileURL); err == nil {
				result = append(result, mapDocumentToAccount(factory, session, document))
				continue
			}

			result = append(result, model.RemoteActorAccount(profileURL, following.Label, following.IconURL, time.UnixMilli(following.CreateDate)))
		}

		return result, toot.PageInfo{}, nil
	}
}

// GetAccount_Endorsements handles the real Mastodon client request shape
// (/api/v1/accounts/:id/endorsements) for accounts featured on a profile.
// Emissary does not (currently?) publish endorsements/featured accounts.
func GetAccount_Endorsements(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Endorsements) ([]object.Account, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetAccount_Endorsements) ([]object.Account, toot.PageInfo, error) {
		return []object.Account{}, toot.PageInfo{}, nil
	}
}

// GetAccount_FeaturedTags implements the Mastodon "get account featured tags" endpoint, and always returns an empty list
func GetAccount_FeaturedTags(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_FeaturedTags) ([]object.Tag, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetAccount_FeaturedTags) ([]object.Tag, toot.PageInfo, error) {

		// Emissary does not (currently?) publish featured tags
		return []object.Tag{}, toot.PageInfo{}, nil
	}
}

// PostAccount_Follow implements the Mastodon "follow account" endpoint
func PostAccount_Follow(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Follow) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Follow"

	return func(auth model.Authorization, t txn.PostAccount_Follow) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Create a new "Following" record
		followingService := factory.Following()
		following := model.NewFollowing()
		following.UserID = auth.UserID

		followingURL, err := resolveAccountURL(factory, session, t.ID)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Resolving account", t.ID)
		}

		following.URL = followingURL

		// Save the record and begin following the remote user.
		if err := followingService.Save(session, &following, "Created via Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Saving following")
		}

		// Return the "Following" record as a Toot. Relationship.ID must match
		// whatever Account.ID the caller already knows this account by -- echo
		// back t.ID verbatim rather than following.Toot()'s own ID (the target's
		// raw ProfileURL), so it stays consistent whether the caller used a local
		// short ID or an actor URL.
		result := following.Toot()
		result.ID = t.ID
		return result, nil
	}
}

// PostAccount_Unfollow implements the Mastodon "unfollow account" endpoint
func PostAccount_Unfollow(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Unfollow) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Unfollow"

	return func(auth model.Authorization, t txn.PostAccount_Unfollow) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the "Following" record
		followingService := factory.Following()
		following := model.NewFollowing()

		followingURL, err := resolveAccountURL(factory, session, t.ID)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Resolving account", t.ID)
		}

		if err := followingService.LoadByURL(session, auth.UserID, followingURL, &following); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Loading following")
		}

		// Delete the "Following" record
		if err := followingService.Delete(session, &following, "Deleted by Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Deleting following")
		}

		// See the comment in PostAccount_Follow -- Relationship.ID echoes t.ID.
		result := following.Toot()
		result.ID = t.ID
		return result, nil
	}
}

// PostAccount_Block implements the Mastodon "block account" endpoint
func PostAccount_Block(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Block) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Block"

	return func(auth model.Authorization, t txn.PostAccount_Block) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Create a new Rule record
		// RULE: A Mastodon "block" is a BLOCK, not NewRule()'s MUTE default
		ruleService := factory.Rule()
		rule := model.NewRule()
		rule.UserID = auth.UserID
		rule.Type = model.RuleTypeActor
		rule.Action = model.RuleActionBlock
		rule.Trigger = t.ID

		if err := ruleService.Save(session, &rule, "Created via Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Saving rule")
		}

		// Return the Rule record as a Toot
		return rule.Toot(), nil
	}
}

// PostAccount_Unblock implements the Mastodon "unblock account" endpoint
func PostAccount_Unblock(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Unblock) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Unblock"

	return func(auth model.Authorization, t txn.PostAccount_Unblock) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Locate the rule record
		ruleService := factory.Rule()
		rule := model.NewRule()

		if err := ruleService.LoadByMatchKey(session, auth.UserID, model.RuleTypeActor, t.ID, &rule); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Loading rule")
		}

		// Delete the rule record
		if err := ruleService.Delete(session, &rule, "Deleted by Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Deleting rule")
		}

		// Return success
		return rule.Toot(), nil
	}
}

// PostAccount_Mute implements the Mastodon "mute account" endpoint
func PostAccount_Mute(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Mute) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Mute"

	return func(auth model.Authorization, t txn.PostAccount_Mute) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Create a new Rule record
		ruleService := factory.Rule()
		rule := model.NewRule()
		rule.UserID = auth.UserID
		rule.Type = model.RuleTypeActor
		rule.Trigger = t.ID

		if err := ruleService.Save(session, &rule, "Created via Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Saving rule")
		}

		// Return the Rule record as a Toot
		return rule.Toot(), nil
	}
}

// PostAccount_Unmute implements the Mastodon "unmute account" endpoint
func PostAccount_Unmute(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Unmute) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Unmute"

	return func(auth model.Authorization, t txn.PostAccount_Unmute) (object.Relationship, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Locate the rule record
		ruleService := factory.Rule()
		rule := model.NewRule()

		if err := ruleService.LoadByMatchKey(session, auth.UserID, model.RuleTypeActor, t.ID, &rule); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Loading rule")
		}

		// Delete the rule record
		if err := ruleService.Delete(session, &rule, "Deleted by Mastodon API"); err != nil {
			return object.Relationship{}, derp.Wrap(err, location, "Deleting rule")
		}

		// Return success
		return rule.Toot(), nil
	}
}

// PostAccount_Pin is the Mastodon "pin account" endpoint, which Emissary does not implement
func PostAccount_Pin(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Pin) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Pin"

	return func(auth model.Authorization, t txn.PostAccount_Pin) (object.Relationship, error) {
		return object.Relationship{}, derp.NotImplemented(location)
	}
}

// PostAccount_Unpin is the Mastodon "unpin account" endpoint, which Emissary does not implement
func PostAccount_Unpin(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Unpin) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Unpin"

	return func(auth model.Authorization, t txn.PostAccount_Unpin) (object.Relationship, error) {
		return object.Relationship{}, derp.NotImplemented(location)
	}
}

// PostAccount_Note is the Mastodon "set private note" endpoint, which Emissary does not implement
func PostAccount_Note(serverFactory *server.Factory) func(model.Authorization, txn.PostAccount_Note) (object.Relationship, error) {

	const location = "handler.mastodon_PostAccount_Note"

	return func(auth model.Authorization, t txn.PostAccount_Note) (object.Relationship, error) {
		return object.Relationship{}, derp.NotImplemented(location)
	}
}

// GetAccount_Relationships implements the Mastodon "check relationships" endpoint.
// The client calls this for every account it shows -- profile screens, follow
// buttons -- and expects a JSON array, so a derp.NotImplemented here breaks those
// screens. Emissary answers "following" from a Following record and
// "blocking"/"muting" from an ACTOR Rule; every other flag is reported false.
func GetAccount_Relationships(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Relationships) ([]object.Relationship, error) {

	const location = "handler.mastodon_GetAccount_Relationships"

	return func(auth model.Authorization, t txn.GetAccount_Relationships) ([]object.Relationship, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		followingService := factory.Following()
		ruleService := factory.Rule()
		result := make([]object.Relationship, 0, len(t.IDs))

		for _, id := range t.IDs {

			relationship := object.Relationship{
				ID:        id,
				Languages: []string{},
			}

			// "following": a Following record for the resolved actor URL.
			if accountURL, err := resolveAccountURL(factory, session, id); err == nil {
				following := model.NewFollowing()
				if err := followingService.LoadByURL(session, auth.UserID, accountURL, &following); err == nil {
					relationship.Following = true
				}
			}

			// "blocking"/"muting": an ACTOR Rule keyed by the ID the client sent,
			// matching how PostAccount_Block / PostAccount_Mute store it.
			rule := model.NewRule()
			if err := ruleService.LoadByMatchKey(session, auth.UserID, model.RuleTypeActor, id, &rule); err == nil {
				switch rule.Action {
				case model.RuleActionBlock:
					relationship.Blocking = true
				case model.RuleActionMute:
					relationship.Muting = true
				}
			}

			result = append(result, relationship)
		}

		return result, nil
	}
}

// GetAccount_FamiliarFollowers is the Mastodon "get familiar followers" endpoint, which Emissary does not implement
func GetAccount_FamiliarFollowers(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_FamiliarFollowers) (object.FamiliarFollowers, error) {

	return func(auth model.Authorization, t txn.GetAccount_FamiliarFollowers) (object.FamiliarFollowers, error) {

		// Emissary doesn't track a followers graph, so an honest empty result is the
		// correct answer for any t.IDs -- same "honest empty" pattern as
		// GetAccount_Followers/GetAccount_Following, rather than derp.NotImplemented.
		return object.FamiliarFollowers{}, nil
	}
}

// https://docs.joinmastodon.org/methods/accounts/#search
func GetAccount_Search(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Search) ([]object.Account, toot.PageInfo, error) {

	const location = "handler.mastodon_GetAccount_Search"

	return func(auth model.Authorization, t txn.GetAccount_Search) ([]object.Account, toot.PageInfo, error) {
		return nil, toot.PageInfo{}, derp.NotImplemented(location)
	}
}

// https://docs.joinmastodon.org/methods/accounts/#lookup
func GetAccount_Lookup(serverFactory *server.Factory) func(model.Authorization, txn.GetAccount_Lookup) (object.Account, error) {

	const location = "handler.mastodon_GetAccount_Lookup"

	return func(auth model.Authorization, t txn.GetAccount_Lookup) (object.Account, error) {

		// Get the factory for this domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the Account as an ActivityStream
		client := factory.ActivityStream().UserClient(auth.UserID)
		document, err := client.Load(t.Acct)

		if err != nil {
			// See the matching comment in GetAccount: don't let a remote origin's own
			// status code read to the client as its own bearer token being invalid.
			return object.Account{}, derp.Wrap(err, location, "Loading document", derp.WithBadGateway())
		}

		// Map the ActivityStream to a Mastodon Account. The caller already told us
		// the exact handle they looked up, which is more authoritative than the
		// "username@host" mapDocumentToAccount would otherwise derive from the
		// document -- so it wins here.
		result := mapDocumentToAccount(factory, session, document)
		result.Acct = t.Acct

		// Success.
		return result, nil
	}
}
