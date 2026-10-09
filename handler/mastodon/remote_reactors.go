package mastodon

import (
	"context"
	"net/url"
	"path"
	"strconv"
	"strings"
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
)

// fetchRemoteReactors asks a post's own server which accounts liked ("favourited_by") or boosted
// ("reblogged_by") it, returning nothing when the server cannot say.
func fetchRemoteReactors(postURL string, ownHost string, listName string, limit int, allowPrivateIPs bool) []object.Account {

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
