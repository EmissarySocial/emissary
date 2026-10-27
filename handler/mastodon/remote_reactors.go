package mastodon

import (
	"context"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/remote"
	"github.com/benpate/toot/object"
	"github.com/benpate/uri"
)

const (
	remoteReactorTimeout     = 6 * time.Second // longest wait for another server's answer
	remoteReactorMaxResponse = 2 << 20         // largest answer accepted, in bytes
	remoteReactorCacheWindow = 2 * time.Minute // how long another server's answer is reused
	remoteReactorCacheLimit  = 1000            // most answers remembered at once
)

// remoteReactorCache remembers recent answers to fetchRemoteReactors
var remoteReactorCache = newReactorMemo(remoteReactorCacheWindow)

// fetchRemoteReactors returns fetchRemoteReactorsLive's answer, reusing one from the last few minutes
func fetchRemoteReactors(postURL string, ownHost string, listName string, limit int, allowPrivateIPs bool) []object.Account {

	// An empty answer is reused too, so a server that cannot say is not asked on every open

	// RULE: Each answer is mapped in one domain's terms, so the domain is part of the key
	key := ownHost + " " + postURL + " " + listName + " " + strconv.Itoa(limit)

	if accounts, found := remoteReactorCache.get(key); found {
		return accounts
	}

	accounts := fetchRemoteReactorsLive(postURL, ownHost, listName, limit, allowPrivateIPs)
	remoteReactorCache.set(key, accounts)

	return accounts
}

// fetchRemoteReactorsLive asks a post's own server which accounts liked ("favourited_by") or boosted
// ("reblogged_by") it, returning nothing when the server cannot say.
func fetchRemoteReactorsLive(postURL string, ownHost string, listName string, limit int, allowPrivateIPs bool) []object.Account {

	// RULE: only a post on another server is asked about; this server already knows its own
	parsed, err := url.Parse(postURL)

	if err != nil || parsed.Host == "" || sameDomain(parsed.Host, ownHost) {
		return nil
	}

	// Find the post's ID, which on Mastodon is the last part of its address
	statusID := path.Base(parsed.Path)

	if statusID == "" || statusID == "." || statusID == "/" {
		return nil
	}

	// ActivityPub publishes only totals, so ask through the Mastodon-compatible public API
	ctx, cancel := context.WithTimeout(context.Background(), remoteReactorTimeout)
	defer cancel()

	accounts := []object.Account{}

	txn := remote.Get(uri.GuessProtocolForHostname(parsed.Host)+parsed.Host+"/api/v1/statuses/"+url.PathEscape(statusID)+"/"+listName).
		Query("limit", strconv.Itoa(limit)).
		Accept("application/json").
		AllowPrivateIPs(allowPrivateIPs).
		MaxResponseSize(remoteReactorMaxResponse).
		WithContext(ctx).
		Result(&accounts)

	if err := txn.Send(); err != nil {
		return nil
	}

	// Rebuild each account in this server's terms
	result := make([]object.Account, 0, len(accounts))

	for _, account := range accounts {
		if mapped, ok := remoteReactorAccount(account, parsed.Host, ownHost); ok {
			result = append(result, mapped)
		}
	}

	return result
}

// remoteReactorAccount rebuilds an account another server described, in this server's terms, reporting
// false for one with no usable profile or one that lives here.
func remoteReactorAccount(account object.Account, originHost string, ownHost string) (object.Account, bool) {

	// RULE: the profile must be a web address on another server
	profile, err := url.Parse(account.URL)

	if err != nil || (profile.Scheme != "https" && profile.Scheme != "http") || profile.Host == "" {
		return object.Account{}, false
	}

	if sameDomain(profile.Host, ownHost) {
		return object.Account{}, false
	}

	// An account on the post's own server is named without a domain; add it
	acct := account.Acct

	if !strings.Contains(acct, "@") {
		acct += "@" + originHost
	}

	// Keep only web addresses for the pictures, and clean the name and bio
	avatar := webURLOrEmpty(account.Avatar)
	header := webURLOrEmpty(account.Header)

	return object.Account{
		ID:             model.EncodeRemoteAccountID(account.URL),
		Username:       account.Username,
		Acct:           acct,
		DisplayName:    convert.SanitizeText(account.DisplayName),
		URL:            account.URL,
		Note:           convert.SanitizeHTML(account.Note),
		Avatar:         avatar,
		AvatarStatic:   avatar,
		Header:         header,
		HeaderStatic:   header,
		Bot:            account.Bot,
		Group:          account.Group,
		Locked:         account.Locked,
		CreatedAt:      account.CreatedAt,
		FollowersCount: account.FollowersCount,
		FollowingCount: account.FollowingCount,
		StatusesCount:  account.StatusesCount,
	}, true
}

// webURLOrEmpty returns the address when it is an http or https URL, and nothing otherwise.
func webURLOrEmpty(address string) string {

	if parsed, err := url.Parse(address); err != nil || parsed == nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return ""
	}

	return address
}

// sameDomain returns TRUE if two hosts name the same domain, ignoring case and any port.
func sameDomain(left string, right string) bool {
	return strings.EqualFold(strings.Split(left, ":")[0], strings.Split(right, ":")[0])
}

// reactorMemo is a small, short-lived, concurrency-safe memory of reactor lists by key
type reactorMemo struct {
	mutex  sync.Mutex
	items  map[string]reactorMemoItem
	window time.Duration
}

// reactorMemoItem is one remembered answer and the time it expires
type reactorMemoItem struct {
	accounts []object.Account
	expires  time.Time
}

// newReactorMemo returns an empty reactorMemo that keeps each answer for the window
func newReactorMemo(window time.Duration) *reactorMemo {
	return &reactorMemo{items: map[string]reactorMemoItem{}, window: window}
}

// get returns a remembered answer that has not expired
func (memo *reactorMemo) get(key string) ([]object.Account, bool) {

	memo.mutex.Lock()
	defer memo.mutex.Unlock()

	item, found := memo.items[key]

	if !found || time.Now().After(item.expires) {
		return nil, false
	}

	return item.accounts, true
}

// set remembers an answer, dropping expired ones first when full, and starting over if that is not enough
func (memo *reactorMemo) set(key string, accounts []object.Account) {

	memo.mutex.Lock()
	defer memo.mutex.Unlock()

	if len(memo.items) >= remoteReactorCacheLimit {

		now := time.Now()

		for itemKey, item := range memo.items {
			if now.After(item.expires) {
				delete(memo.items, itemKey)
			}
		}

		if len(memo.items) >= remoteReactorCacheLimit {
			memo.items = map[string]reactorMemoItem{}
		}
	}

	memo.items[key] = reactorMemoItem{accounts: accounts, expires: time.Now().Add(memo.window)}
}
