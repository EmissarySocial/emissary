package mastodon

import (
	"cmp"
	"net/http"
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
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Mastodon Follow Suggestions
 *
 * Suggests the most-followed public Users on this server
 * who have posted lately, skipping anyone the caller
 * already follows, blocks, or mutes, or who blocks them.
 * https://docs.joinmastodon.org/methods/suggestions/
 ******************************************/

const (
	suggestionsDefault    = 40       // Mastodon's default number of suggestions
	suggestionsMax        = 80       // Mastodon's largest number of suggestions
	suggestionsPoolFactor = 3        // How many candidates to load per suggestion, leaving room to filter
	suggestionsActiveDays = 30       // How recently a User must have posted to be suggested
	suggestionSourceLocal = "global" // Mastodon's source for "popular on this server"
)

// suggestionPick is one User who may be suggested, with the date of their newest public post
type suggestionPick struct {
	user   model.User
	newest int64
}

// suggestionFilter holds what rules a candidate out of the caller's suggestions
type suggestionFilter struct {
	viewerID primitive.ObjectID          // The caller, who is never suggested to themselves
	followed map[string]bool             // Actor URLs the caller already follows
	hidden   map[string]bool             // Match keys of the caller's active BLOCK and MUTE Rules
	blockers map[primitive.ObjectID]bool // Users who block the caller
}

// GetSuggestions implements the Mastodon "follow suggestions" endpoint
func GetSuggestions(serverFactory *server.Factory) func(model.Authorization, txn.GetSuggestions) ([]object.Suggestion, error) {

	const location = "handler.mastodon.GetSuggestions"

	return func(auth model.Authorization, t txn.GetSuggestions) ([]object.Suggestion, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Make new friends, but keep the old
		return suggestions(factory, session, auth, suggestionsLimit(t.Limit))
	}
}

// GetSuggestions_V1 implements the older Mastodon "follow suggestions" endpoint, which lists only the accounts
func GetSuggestions_V1(serverFactory *server.Factory) echo.HandlerFunc {

	// Ice Cubes reads this one; toot has no v1 route, so it is registered in server.go
	const location = "handler.mastodon.GetSuggestions_V1"

	return func(ctx echo.Context) error {

		// RULE: Suggestions are personal, so the caller must be signed in
		auth, err := Authorizer(serverFactory)(ctx.Request())

		if err != nil {
			return derp.Wrap(err, location, "Authorizing request")
		}

		// RULE: The token must grant read access, the same scope toot requires for v2
		if !hasReadScope(auth.Scopes()) {
			return derp.Forbidden(location, "Token lacks the read scope")
		}

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(serverFactory.Hostname(ctx.Request()))

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Find the same suggestions as v2
		limit, _ := strconv.Atoi(ctx.QueryParam("limit")) // A missing or malformed limit falls back to the default
		picks, err := suggestions(factory, session, auth, suggestionsLimit(limit))

		if err != nil {
			return derp.Wrap(err, location, "Finding suggestions")
		}

		// Send just the accounts
		result := make([]object.Account, 0, len(picks))

		for _, pick := range picks {
			result = append(result, pick.Account)
		}

		// People you may know (or should)
		return ctx.JSON(http.StatusOK, result)
	}
}

// suggestions returns up to `limit` active, public Users on this server that the caller might follow, as Mastodon suggestions
func suggestions(factory *service.Factory, session data.Session, auth model.Authorization, limit int) ([]object.Suggestion, error) {

	const location = "handler.mastodon.suggestions"

	// Load the signed-in User
	viewer := model.NewUser()

	if err := factory.User().LoadByID(session, auth.UserID, &viewer); err != nil {
		return nil, derp.Wrap(err, location, "Loading user", auth.UserID)
	}

	// Load the most-followed public Users, with room to filter some out
	candidates, err := factory.User().Query(session,
		exp.Equal("isPublic", true),
		option.SortDesc("followerCount"),
		option.MaxRows(int64(limit*suggestionsPoolFactor)),
	)

	if err != nil {
		return nil, derp.Wrap(err, location, "Querying candidates")
	}

	// Learn who the caller follows, blocks, and mutes, and who blocks them
	filter, err := loadSuggestionFilter(factory, session, &viewer, candidates)

	if err != nil {
		return nil, derp.Wrap(err, location, "Loading follows and rules")
	}

	// RULE: Suggest only Users who posted publicly in the last month
	since := time.Now().AddDate(0, 0, -suggestionsActiveDays).Unix()
	picks := make([]suggestionPick, 0, len(candidates))

	for index := range candidates {

		if !filter.allows(&candidates[index]) {
			continue
		}

		newest, err := factory.Stream().NewestPublicPost(session, candidates[index].UserID)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Finding newest post", candidates[index].UserID))
			continue
		}

		if newest > since {
			picks = append(picks, suggestionPick{user: candidates[index], newest: newest})
		}
	}

	// Rank them, and keep the best
	sortSuggestions(picks)

	if len(picks) > limit {
		picks = picks[:limit]
	}

	// Describe each pick as a Mastodon account, "popular on this server"
	result := make([]object.Suggestion, 0, len(picks))

	for index := range picks {
		result = append(result, object.Suggestion{
			Source:  suggestionSourceLocal,
			Account: tootUser(factory, session, auth, &picks[index].user),
		})
	}

	// You might also like...
	return result, nil
}

// hasReadScope returns TRUE if the scopes grant read access to accounts
func hasReadScope(scopes []string) bool {
	return slices.Contains(scopes, "read") || slices.Contains(scopes, "read:accounts")
}

// DeleteSuggestion implements the Mastodon "remove a suggestion" endpoint as a no-op
func DeleteSuggestion(serverFactory *server.Factory) func(model.Authorization, txn.DeleteSuggestion) (struct{}, error) {

	return func(model.Authorization, txn.DeleteSuggestion) (struct{}, error) {
		// Emissary keeps no list of dismissed suggestions, so a dismissed account may return later
		return struct{}{}, nil
	}
}

// suggestionsLimit returns the number of suggestions to send, within Mastodon's bounds
func suggestionsLimit(limit int) int {

	if limit <= 0 {
		return suggestionsDefault
	}

	return min(limit, suggestionsMax)
}

// loadSuggestionFilter returns the caller's follows, blocks, and mutes that concern the candidates,
// and which candidates block the caller
func loadSuggestionFilter(factory *service.Factory, session data.Session, viewer *model.User, candidates []model.User) (suggestionFilter, error) {

	const location = "handler.mastodon.loadSuggestionFilter"

	// Start with nothing ruled out, apart from the caller
	filter := suggestionFilter{
		viewerID: viewer.UserID,
		followed: map[string]bool{},
		hidden:   map[string]bool{},
		blockers: map[primitive.ObjectID]bool{},
	}

	// No candidates, no questions
	if len(candidates) == 0 {
		return filter, nil
	}

	// Collect each candidate's ID, actor URL, and Rule match keys, so each question below is one query
	candidateIDs := make([]primitive.ObjectID, 0, len(candidates))
	urls := make([]string, 0, len(candidates))
	keys := make([]string, 0, len(candidates)*2)

	for index := range candidates {
		candidateIDs = append(candidateIDs, candidates[index].UserID)
		urls = append(urls, candidates[index].ActivityPubURL())
		keys = append(keys, model.ActorMatchKeys(candidates[index].ActivityPubURL())...)
	}

	// Find the candidates the caller already follows
	followings, err := factory.Following().Query(session,
		exp.Equal("userId", viewer.UserID).AndIn("profileUrl", urls),
		option.Fields("profileUrl"),
	)

	if err != nil {
		return filter, derp.Wrap(err, location, "Querying follows")
	}

	// Remember them by actor URL
	for _, following := range followings {
		filter.followed[following.ProfileURL] = true
	}

	// Find the caller's (and the admin's) Rules that cover a candidate or their server
	rules, err := factory.Rule().QueryByMatchKeys(session, viewer.UserID, keys)

	if err != nil {
		return filter, derp.Wrap(err, location, "Querying rules")
	}

	// Remember the unexpired BLOCK and MUTE Rules by match key; LABEL Rules hide no one
	filter.hidden = hidingMatchKeys(rules, time.Now().Unix())

	// Find the candidates who block the caller
	blocks, err := factory.Rule().QuerySummary(session,
		exp.In("userId", candidateIDs).
			AndEqual("matchKey", model.RuleMatchKey(model.RuleTypeActor, viewer.ActivityPubURL())).
			AndEqual("action", model.RuleActionBlock),
	)

	if err != nil {
		return filter, derp.Wrap(err, location, "Querying blocks")
	}

	// Remember who blocks the caller; a mute does not stop the caller following them
	for _, block := range activeRules(blocks, time.Now().Unix()) {
		filter.blockers[block.UserID] = true
	}

	// Filter, assembled
	return filter, nil
}

// allows returns TRUE if the User may be suggested to the caller
func (filter suggestionFilter) allows(user *model.User) bool {

	// RULE: Never suggest the caller, an account that has moved, or anyone already followed
	if user.UserID == filter.viewerID || user.MovedTo != "" || filter.followed[user.ActivityPubURL()] {
		return false
	}

	// RULE: Never suggest anyone who blocks the caller
	if filter.blockers[user.UserID] {
		return false
	}

	// RULE: Never suggest anyone the caller blocks or mutes, by account or by server
	for _, key := range model.ActorMatchKeys(user.ActivityPubURL()) {
		if filter.hidden[key] {
			return false
		}
	}

	// Come on in
	return true
}

// hidingMatchKeys returns the match keys of the unexpired BLOCK and MUTE Rules in the list
func hidingMatchKeys(rules []model.RuleSummary, now int64) map[string]bool {

	result := map[string]bool{}

	for _, rule := range activeRules(rules, now) {
		if rule.Action == model.RuleActionBlock || rule.Action == model.RuleActionMute {
			result[rule.MatchKey] = true
		}
	}

	return result
}

// activeRules returns the Rules in the list that have not expired
func activeRules(rules []model.RuleSummary, now int64) []model.RuleSummary {

	return slices.DeleteFunc(slices.Clone(rules), func(rule model.RuleSummary) bool {
		return rule.ExpireDate > 0 && rule.ExpireDate < now
	})
}

// sortSuggestions puts the most-followed Users first, breaking ties by who posted most recently
func sortSuggestions(picks []suggestionPick) {

	slices.SortStableFunc(picks, func(a, b suggestionPick) int {
		return cmp.Or(
			cmp.Compare(b.user.FollowerCount, a.user.FollowerCount),
			cmp.Compare(b.newest, a.newest),
		)
	})
}
