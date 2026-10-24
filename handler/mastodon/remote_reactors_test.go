package mastodon

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

// reactorJSON is a Mastodon-style list of accounts, with one on this server, one without a usable profile, and unsafe values.
const reactorJSON = `[
	{"id":"110088","username":"Gina","acct":"Gina","display_name":"Gina <b>G</b>","url":"https://fosstodon.org/@Gina","avatar":"https://cdn.example.com/g.png","header":"javascript:alert(1)","note":"<p>hi</p><script>x</script>","created_at":"2019-03-26T00:00:00.000Z"},
	{"id":"2","username":"ElyseMGrasso","acct":"ElyseMGrasso@wandering.shop","display_name":"Elyse","url":"https://wandering.shop/@ElyseMGrasso"},
	{"id":"3","username":"mine","acct":"mine@my.example","display_name":"Mine","url":"https://my.example/@abc"},
	{"id":"4","username":"nourl","acct":"nourl","display_name":"No URL","url":"javascript:alert(1)"}
]`

// TestFetchRemoteReactors_ReadsAndRebuildsTheOtherServersList serves a Mastodon-style list and checks each
// account is rebuilt in this server's terms, and that accounts which cannot be used are dropped.
func TestFetchRemoteReactors_ReadsAndRebuildsTheOtherServersList(t *testing.T) {

	var requested string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reactorJSON))
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	accounts := fetchRemoteReactors(server.URL+"/users/davep/statuses/116811463127343333", "my.example", "favourited_by", 40, true)

	require.Equal(t, "/api/v1/statuses/116811463127343333/favourited_by?limit=40", requested)
	require.Len(t, accounts, 2, "an account that lives here, and one with no usable profile, are left out")

	gina := accounts[0]
	require.Equal(t, "Gina@"+host, gina.Acct, "an account on the post's own server gets that server's domain")
	require.Equal(t, model.EncodeRemoteAccountID("https://fosstodon.org/@Gina"), gina.ID)
	require.Equal(t, "Gina G", gina.DisplayName, "markup is stripped from names")
	require.Equal(t, "https://cdn.example.com/g.png", gina.Avatar)
	require.Equal(t, "", gina.Header, "an address that is not a web address is dropped")
	require.NotContains(t, gina.Note, "<script")

	require.Equal(t, "ElyseMGrasso@wandering.shop", accounts[1].Acct, "a full handle is kept as it is")
}

// TestFetchRemoteReactors_YieldsNothingWhenTheServerCannotHelp covers an error response, a post on this server, and a bad address.
func TestFetchRemoteReactors_YieldsNothingWhenTheServerCannotHelp(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()

	require.Empty(t, fetchRemoteReactors(server.URL+"/users/x/statuses/1", "my.example", "favourited_by", 40, true))

	host := strings.TrimPrefix(server.URL, "http://")
	require.Empty(t, fetchRemoteReactors(server.URL+"/users/x/statuses/1", host, "favourited_by", 40, true), "a post on this server is never asked about")
	require.Empty(t, fetchRemoteReactors("not a url", "my.example", "favourited_by", 40, true))
}

// TestMergeReactors_SkipsAnyoneListedTwice confirms the same person is not shown twice, ignoring case.
func TestMergeReactors_SkipsAnyoneListedTwice(t *testing.T) {

	known := []object.Account{{Acct: "ben@mastodon.social"}}
	others := []object.Account{{Acct: "Ben@Mastodon.Social"}, {Acct: "gina@fosstodon.org"}, {Acct: "gina@fosstodon.org"}}

	merged := mergeReactors(known, others)

	require.Len(t, merged, 2)
	require.Equal(t, "gina@fosstodon.org", merged[1].Acct)
}
