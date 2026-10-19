package mastodon

import (
	"context"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/remote"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	hashtagMaxHosts    = 8
	hashtagHostTimeout = 6 * time.Second
	hashtagMaxResponse = 4 << 20

	hashtagHistoryDays  = 7  // Days of history the client sums into a hashtag's stats
	hashtagHistoryFetch = 40 // Posts requested from each server to tally that history (Mastodon's page maximum)

	hashtagLocalHistoryMax = 500 // Most of this server's own posts tallied into a tag's history

	hashtagFeedScan    = 200             // Most recent feed items checked for a hashtag
	hashtagFeedTimeout = 4 * time.Second // Longest the feed scan may take before it returns what it has
)

// https://docs.joinmastodon.org/methods/timelines/#tag
//
// Emissary indexes no posts by hashtag, so this asks the servers of the accounts
// the caller follows for their public posts under the tag, and merges the results.
// Those servers hand back their own IDs, so every status is remapped to the IDs
// this API uses everywhere else (see remapRemoteStatus).
//
// The remote servers' cursors don't map onto max_id/min_id, so this returns a
// single page, newest first, with no paging info.
func GetTimeline_Hashtag(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_Hashtag) ([]object.Status, toot.PageInfo, error) {

	const location = "handler.mastodon.GetTimeline_Hashtag"

	return func(auth model.Authorization, t txn.GetTimeline_Hashtag) ([]object.Status, toot.PageInfo, error) {

		hashtag := strings.TrimPrefix(strings.TrimSpace(t.Hashtag), "#")

		if hashtag == "" {
			return []object.Status{}, toot.PageInfo{}, nil
		}

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Invalid Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		limit := int(pageLimit(t.Limit))

		// The feed scan and the remote requests both spend their time waiting, so run them together
		var feed []object.Status

		feedDone := make(chan struct{})

		go func() {
			defer close(feedDone)
			feed = feedHashtagStatuses(factory, session, auth, hashtag, limit)
		}()

		results, err := fetchHashtagResults(factory, session, auth.UserID, t.Host, hashtag, limit)

		if err != nil {
			<-feedDone
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Fetching hashtag posts")
		}

		remote := mergeHashtagStatuses(results, limit, t.OnlyMedia)

		// Local posts and the caller's own feed come first, so they win over a remote copy
		own := localHashtagStatuses(factory, session, auth, hashtag, int64(limit), exp.All())
		markReacted(factory, session, auth.UserID, own)
		markBookmarked(factory, session, auth.UserID, own)

		<-feedDone
		own = append(own, feed...)

		return combineHashtagStatuses(own, remote, limit, t.OnlyMedia), toot.PageInfo{}, nil
	}
}

// https://docs.joinmastodon.org/methods/tags/#get
//
// Emissary has no local index of posts by hashtag, so history comes from followed servers
// (mergeHashtagHistories), falling back to a tally of the posts they show if none publish one.
func GetTag(serverFactory *server.Factory) func(model.Authorization, txn.GetTag) (object.Tag, error) {

	const location = "handler.mastodon.GetTag"

	return func(auth model.Authorization, t txn.GetTag) (object.Tag, error) {

		name := strings.TrimPrefix(strings.TrimSpace(t.ID), "#")

		result := object.Tag{
			Name:    name,
			URL:     uri.GuessProtocolForHostname(t.Host) + t.Host + "/tags/" + url.PathEscape(name),
			History: []object.TagHistory{},
		}

		if name == "" || !auth.IsAuthenticated() {
			return result, nil
		}

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			derp.Report(err)
			return result, nil
		}

		defer cancel()

		histories, err := fetchHashtagHistories(factory, session, auth.UserID, t.Host, name)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Fetching hashtag history"))
			return result, nil
		}

		// This server's own posts are one more source of history
		since := time.Now().AddDate(0, 0, -hashtagHistoryDays).Unix()
		local := localHashtagStatuses(factory, session, auth, name, hashtagLocalHistoryMax, exp.GreaterThan("publishDate", since))

		if localHistory := hashtagHistory(local, time.Now(), hashtagHistoryDays); len(local) > 0 {
			histories = append(histories, localHistory)
		}

		if merged := mergeHashtagHistories(histories, hashtagHistoryDays); len(merged) > 0 {
			result.History = merged
			return result, nil
		}

		// No server published a history, so tally the posts they do show
		results, err := fetchHashtagResults(factory, session, auth.UserID, t.Host, name, hashtagHistoryFetch)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Fetching hashtag posts"))
			return result, nil
		}

		statuses := mergeHashtagStatuses(results, hashtagHistoryFetch*hashtagMaxHosts, false)
		result.History = hashtagHistory(statuses, time.Now(), hashtagHistoryDays)
		return result, nil
	}
}

// localHashtagStatuses returns this server's own posts tagged with a hashtag, as the caller may see them.
// A failure is reported and yields nothing, so the remote results still show.
func localHashtagStatuses(factory *service.Factory, session data.Session, auth model.Authorization, hashtag string, limit int64, criteria exp.Expression) []object.Status {

	const location = "handler.mastodon.localHashtagStatuses"

	// Tag names keep whatever spelling was first used, so try the likely ones
	names := []string{hashtag, strings.ToLower(hashtag)}

	if known, err := factory.SearchTag().QueryByValue(session, []string{model.ToToken(hashtag)}); err == nil {
		for _, tag := range known {
			names = append(names, tag.Name)
		}
	}

	streams, err := factory.Stream().QueryByHashtag(session, auth, names, criteria, option.MaxRows(limit))

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Querying local posts", hashtag))
		return nil
	}

	result := make([]object.Status, 0, len(streams))

	for index := range streams {

		// RULE: the query matches any tag with that name, including @mentions
		if !slices.ContainsFunc(model.TagNames(streams[index].Tags, vocab.LinkTypeHashtag), func(name string) bool {
			return strings.EqualFold(name, hashtag)
		}) {
			continue
		}

		result = append(result, tootStream(factory, session, &streams[index]))
	}

	return result
}

// feedHashtagStatuses returns posts in the caller's own feed carrying a hashtag, catching any
// software (not just Mastodon-API servers) and keeping each post's feed ID for favourite/boost.
func feedHashtagStatuses(factory *service.Factory, session data.Session, auth model.Authorization, hashtag string, limit int) []object.Status {

	const location = "handler.mastodon.feedHashtagStatuses"

	if !auth.IsAuthenticated() {
		return nil
	}

	items, err := factory.NewsFeed().QueryByUserID(session, auth.UserID, exp.All(), option.SortDesc("rank"), option.MaxRows(hashtagFeedScan))

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Querying feed", hashtag))
		return nil
	}

	const maxConcurrent = 16

	client := factory.ActivityStream().UserClient(auth.UserID)

	var mutex sync.Mutex
	matched := make([]model.NewsItem, 0)

	done := make(chan struct{})

	go func() {

		defer close(done)

		var waitGroup sync.WaitGroup
		slots := make(chan struct{}, maxConcurrent)

		for _, item := range items {

			waitGroup.Add(1)
			slots <- struct{}{}

			go func(item model.NewsItem) {
				defer waitGroup.Done()
				defer func() { <-slots }()

				document, err := client.Load(item.URL)

				if err != nil || !documentHasHashtag(document, hashtag) {
					return
				}

				mutex.Lock()
				matched = append(matched, item)
				mutex.Unlock()
			}(item)
		}

		waitGroup.Wait()
	}()

	select {
	case <-done:
	case <-time.After(hashtagFeedTimeout):
	}

	// Copy under the lock: a slow lookup may still be finishing
	mutex.Lock()
	found := slices.Clone(matched)
	mutex.Unlock()

	slices.SortStableFunc(found, func(a, b model.NewsItem) int {
		return int(b.Rank - a.Rank)
	})

	if len(found) > limit {
		found = found[:limit]
	}

	return newsItemsToPosts(factory, session, auth, found)
}

// documentHasHashtag returns TRUE if a post document is tagged with the hashtag (in any capitalization).
func documentHasHashtag(document streams.Document, hashtag string) bool {

	return slices.ContainsFunc(mapDocumentToTags(document), func(tag object.StatusTag) bool {
		return strings.EqualFold(tag.Name, hashtag)
	})
}

// combineHashtagStatuses merges this server's posts into the remote results. A post this
// server has, and a remote copy of it, are one post -- the local one wins, since it keeps
// this API's own ID.
func combineHashtagStatuses(local []object.Status, remote []object.Status, limit int, onlyMedia bool) []object.Status {

	seen := map[string]bool{}
	merged := make([]object.Status, 0, len(local)+len(remote))

	for _, status := range append(local, remote...) {

		if seen[status.URI] || (onlyMedia && len(status.MediaAttachments) == 0) {
			continue
		}

		seen[status.URI] = true
		merged = append(merged, status)
	}

	slices.SortStableFunc(merged, func(a, b object.Status) int {
		return strings.Compare(b.CreatedAt, a.CreatedAt)
	})

	if len(merged) > limit {
		merged = merged[:limit]
	}

	return merged
}

// hashtagHosts returns the servers of the accounts the caller follows, other than this one.
func hashtagHosts(factory *service.Factory, session data.Session, userID primitive.ObjectID, ownHost string) ([]string, error) {

	followings, err := factory.Following().RangeByUserID(session, userID)

	if err != nil {
		return nil, derp.Wrap(err, "handler.mastodon.hashtagHosts", "Listing followings")
	}

	profileURLs := make([]string, 0)

	for following := range followings {
		profileURLs = append(profileURLs, following.ProfileURL)
	}

	return hostsOf(profileURLs, ownHost), nil
}

// fetchHashtagResults asks the servers of the accounts the caller follows for their
// public posts under a hashtag, one result set per server.
func fetchHashtagResults(factory *service.Factory, session data.Session, userID primitive.ObjectID, ownHost string, hashtag string, limit int) ([][]object.Status, error) {

	hosts, err := hashtagHosts(factory, session, userID, ownHost)

	if err != nil {
		return nil, derp.Wrap(err, "handler.mastodon.fetchHashtagResults", "Listing hosts")
	}

	allowPrivateIPs := factory.ActivityStream().AllowPrivateIPs()
	results := make([][]object.Status, len(hosts))

	var waitGroup sync.WaitGroup

	for index, host := range hosts {

		waitGroup.Add(1)

		go func(index int, host string) {
			defer waitGroup.Done()
			results[index] = fetchHashtagStatuses(host, hashtag, limit, allowPrivateIPs)
		}(index, host)
	}

	waitGroup.Wait()
	return results, nil
}

// fetchHashtagHistories asks each server for its own published usage history of a hashtag.
func fetchHashtagHistories(factory *service.Factory, session data.Session, userID primitive.ObjectID, ownHost string, hashtag string) ([][]object.TagHistory, error) {

	hosts, err := hashtagHosts(factory, session, userID, ownHost)

	if err != nil {
		return nil, derp.Wrap(err, "handler.mastodon.fetchHashtagHistories", "Listing hosts")
	}

	allowPrivateIPs := factory.ActivityStream().AllowPrivateIPs()
	results := make([][]object.TagHistory, len(hosts))

	var waitGroup sync.WaitGroup

	for index, host := range hosts {

		waitGroup.Add(1)

		go func(index int, host string) {
			defer waitGroup.Done()
			results[index] = fetchHashtagHistory(host, hashtag, allowPrivateIPs)
		}(index, host)
	}

	waitGroup.Wait()
	return results, nil
}

// fetchHashtagHistory reads one server's usage history for a hashtag, using the
// SSRF-guarded client. Any failure (a server that requires sign-in, one that isn't
// Mastodon-compatible, a timeout) yields nothing.
func fetchHashtagHistory(host string, hashtag string, allowPrivateIPs bool) []object.TagHistory {

	ctx, cancel := context.WithTimeout(context.Background(), hashtagHostTimeout)
	defer cancel()

	tag := object.Tag{}

	txn := remote.Get(uri.GuessProtocolForHostname(host) + host + "/api/v1/tags/" + url.PathEscape(hashtag)).
		Accept("application/json").
		AllowPrivateIPs(allowPrivateIPs).
		MaxResponseSize(hashtagMaxResponse).
		WithContext(ctx).
		Result(&tag)

	if err := txn.Send(); err != nil {
		return nil
	}

	return tag.History
}

// mergeHashtagHistories combines several servers' histories into one, newest day first.
// RULE: takes the largest per-day figure, never the sum -- the same post reaches many servers.
func mergeHashtagHistories(histories [][]object.TagHistory, days int) []object.TagHistory {

	type tally struct{ uses, accounts int }

	perDay := map[int64]tally{}

	for _, history := range histories {
		for _, entry := range history {

			day, err := strconv.ParseInt(entry.Day, 10, 64)

			if err != nil {
				continue
			}

			current := perDay[day]
			current.uses = max(current.uses, atoiOrZero(entry.Uses))
			current.accounts = max(current.accounts, atoiOrZero(entry.Accounts))
			perDay[day] = current
		}
	}

	dayList := make([]int64, 0, len(perDay))

	for day := range perDay {
		dayList = append(dayList, day)
	}

	slices.SortFunc(dayList, func(a, b int64) int { return int(b - a) })

	if len(dayList) > days {
		dayList = dayList[:days]
	}

	result := make([]object.TagHistory, 0, len(dayList))

	for _, day := range dayList {
		result = append(result, object.TagHistory{
			Day:      strconv.FormatInt(day, 10),
			Uses:     strconv.Itoa(perDay[day].uses),
			Accounts: strconv.Itoa(perDay[day].accounts),
		})
	}

	return result
}

// hashtagHistory tallies statuses into Mastodon's per-day tag history: one entry for each
// of the last `days` UTC days, newest first (the client reads the first entry as "today"),
// counting posts and the distinct accounts that made them.
func hashtagHistory(statuses []object.Status, now time.Time, days int) []object.TagHistory {

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	uses := make([]int, days)
	accounts := make([]map[string]bool, days)

	for index := range accounts {
		accounts[index] = map[string]bool{}
	}

	for _, status := range statuses {

		createdAt, err := time.Parse(time.RFC3339Nano, status.CreatedAt)

		if err != nil {
			continue
		}

		createdAt = createdAt.UTC()
		createdDay := time.Date(createdAt.Year(), createdAt.Month(), createdAt.Day(), 0, 0, 0, 0, time.UTC)
		index := int(today.Sub(createdDay).Hours() / 24)

		if index < 0 || index >= days {
			continue
		}

		uses[index]++
		accounts[index][status.Account.URL] = true
	}

	result := make([]object.TagHistory, days)

	for index := range result {
		result[index] = object.TagHistory{
			Day:      strconv.FormatInt(today.AddDate(0, 0, -index).Unix(), 10),
			Uses:     strconv.Itoa(uses[index]),
			Accounts: strconv.Itoa(len(accounts[index])),
		}
	}

	return result
}

// hostsOf returns the distinct hostnames of the provided URLs, skipping this
// server's own and capping the count so one request can't fan out without limit.
func hostsOf(urls []string, ownHost string) []string {

	result := make([]string, 0, hashtagMaxHosts)

	for _, value := range urls {

		parsed, err := url.Parse(value)

		if err != nil || parsed.Host == "" || parsed.Host == ownHost {
			continue
		}

		if slices.Contains(result, parsed.Host) {
			continue
		}

		result = append(result, parsed.Host)

		if len(result) >= hashtagMaxHosts {
			break
		}
	}

	return result
}

// fetchHashtagStatuses asks one server for its public posts under a hashtag,
// using the SSRF-guarded client. Any failure (a server that requires sign-in for
// public timelines, one that isn't Mastodon-compatible, a timeout) yields nothing.
func fetchHashtagStatuses(host string, hashtag string, limit int, allowPrivateIPs bool) []object.Status {

	ctx, cancel := context.WithTimeout(context.Background(), hashtagHostTimeout)
	defer cancel()

	statuses := make([]object.Status, 0)

	txn := remote.Get(uri.GuessProtocolForHostname(host)+host+"/api/v1/timelines/tag/"+url.PathEscape(hashtag)).
		Query("limit", strconv.Itoa(limit)).
		Accept("application/json").
		AllowPrivateIPs(allowPrivateIPs).
		MaxResponseSize(hashtagMaxResponse).
		WithContext(ctx).
		Result(&statuses)

	if err := txn.Send(); err != nil {
		return nil
	}

	return statuses
}

// mergeHashtagStatuses remaps every server's statuses, drops duplicates (the same
// post reaches us from several servers), and returns the newest `limit` of them.
func mergeHashtagStatuses(results [][]object.Status, limit int, onlyMedia bool) []object.Status {

	seen := map[string]bool{}
	merged := make([]object.Status, 0)

	for _, statuses := range results {
		for _, status := range statuses {

			status, ok := remapRemoteStatus(status)

			if !ok || seen[status.URI] {
				continue
			}

			if onlyMedia && len(status.MediaAttachments) == 0 {
				continue
			}

			seen[status.URI] = true
			merged = append(merged, status)
		}
	}

	// Timestamps are ISO 8601 UTC in one fixed format, so they sort as strings
	slices.SortStableFunc(merged, func(a, b object.Status) int {
		return strings.Compare(b.CreatedAt, a.CreatedAt)
	})

	if len(merged) > limit {
		merged = merged[:limit]
	}

	return merged
}

// remapRemoteStatus converts a status fetched from another Mastodon server into
// one this API can serve. ok is false for anything that can't be used.
//
// RULE: nothing from a remote server may be trusted as-is. IDs are replaced with
// the encoded URLs used everywhere else (the remote's IDs mean nothing here), HTML
// is sanitized, and the viewer-specific flags -- which describe the anonymous
// request that fetched it, not the caller -- are cleared.
func remapRemoteStatus(status object.Status) (object.Status, bool) {

	// A boost has no content of its own, and a post needs URLs to be addressable
	if status.Reblog != nil || status.URI == "" || status.Account.URL == "" {
		return status, false
	}

	status.ID = model.EncodeRemoteStatusID(status.URI)
	status.Content = convert.SanitizeHTML(status.Content)
	status.MediaAttachments = safeMediaAttachments(status.MediaAttachments)

	status.Account = remapRemoteAccount(status.Account)

	mentions := make([]object.StatusMention, 0, len(status.Mentions))

	for _, mention := range status.Mentions {

		if mention.URL == "" {
			continue
		}

		mention.ID = model.EncodeRemoteAccountID(mention.URL)
		mention.Acct = acctWithHost(mention.Acct, mention.URL)
		mentions = append(mentions, mention)
	}

	status.Mentions = mentions

	// These IDs belong to the remote server, and so do polls, cards, and the app
	status.InReplyToID = ""
	status.InReplyToAccountID = ""
	status.Poll = nil
	status.Card = nil
	status.Application = nil
	status.Quote = nil
	status.QuoteApproval = nil

	status.Favourited = false
	status.Reblogged = false
	status.Bookmarked = false
	status.Muted = false
	status.Pinned = false

	return status, true
}

// safeMediaAttachments keeps only the media the client can lay out.
//
// RULE: the client sizes an image grid by averaging its attachments' aspect
// ratios, so a visual attachment with no width/height drops out of the average,
// and a post whose attachments all lack them divides by zero and crashes the app.
// A remote server can send anything, so anything unsized is dropped, as is any
// type the client has no renderer for.
func safeMediaAttachments(attachments []object.MediaAttachment) []object.MediaAttachment {

	result := make([]object.MediaAttachment, 0, len(attachments))

	for _, attachment := range attachments {

		switch attachment.Type {

		case "audio":
			// Audio is not laid out in a grid

		case "image", "gifv", "video":

			original, _ := attachment.Meta["original"].(map[string]any)

			if convertToInt(original["width"]) <= 0 || convertToInt(original["height"]) <= 0 {
				continue
			}

		default:
			continue
		}

		if attachment.URL == "" {
			continue
		}

		result = append(result, attachment)
	}

	return result
}

// convertToInt reads a JSON number (decoded as float64) or an int as an int
func convertToInt(value any) int {

	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	}

	return 0
}

// remapRemoteAccount gives an account from another server this API's ID and a
// fully-qualified acct, and sanitizes its HTML.
func remapRemoteAccount(account object.Account) object.Account {

	account.ID = model.EncodeRemoteAccountID(account.URL)
	account.Acct = acctWithHost(account.Acct, account.URL)
	account.Note = convert.SanitizeHTML(account.Note)

	return account
}

// acctWithHost qualifies an acct with its server. A server reports its own accounts
// as a bare username, which would be ambiguous here.
func acctWithHost(acct string, profileURL string) string {

	if strings.Contains(acct, "@") {
		return acct
	}

	parsed, err := url.Parse(profileURL)

	if err != nil || parsed.Hostname() == "" {
		return acct
	}

	return acct + "@" + parsed.Hostname()
}

// atoiOrZero reads a Mastodon count, which the API sends as a string, treating anything unparseable as 0.
func atoiOrZero(value string) int {
	number, _ := strconv.Atoi(value)
	return number
}
