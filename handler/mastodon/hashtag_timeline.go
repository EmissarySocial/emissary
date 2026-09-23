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
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"github.com/benpate/uri"
)

const (
	hashtagMaxHosts    = 8
	hashtagHostTimeout = 6 * time.Second
	hashtagMaxResponse = 4 << 20
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

		// The servers to ask are the ones the caller already follows accounts on
		followings, err := factory.Following().RangeByUserID(session, auth.UserID)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Listing followings")
		}

		profileURLs := make([]string, 0)

		for following := range followings {
			profileURLs = append(profileURLs, following.ProfileURL)
		}

		hosts := hostsOf(profileURLs, t.Host)
		limit := int(pageLimit(t.Limit))
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

		statuses := mergeHashtagStatuses(results, limit, t.OnlyMedia)
		return statuses, toot.PageInfo{}, nil
	}
}

// https://docs.joinmastodon.org/methods/tags/#get
func GetTag(serverFactory *server.Factory) func(model.Authorization, txn.GetTag) (object.Tag, error) {

	return func(auth model.Authorization, t txn.GetTag) (object.Tag, error) {

		name := strings.TrimPrefix(strings.TrimSpace(t.ID), "#")

		return object.Tag{
			Name:    name,
			URL:     uri.GuessProtocolForHostname(t.Host) + t.Host + "/tags/" + url.PathEscape(name),
			History: []object.TagHistory{},
		}, nil
	}
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
