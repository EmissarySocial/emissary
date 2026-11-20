package mastodon

import (
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Mastodon Filters
 *
 * Each Filter is one of the User's hashtag (TAG) Rules.
 * https://docs.joinmastodon.org/methods/filters/
 ******************************************/

// GetFilters implements the Mastodon "list filters" endpoint, showing the User's hashtag Rules
func GetFilters(serverFactory *server.Factory) func(model.Authorization, txn.GetFilters) ([]object.Filter, error) {

	const location = "handler.mastodon.GetFilters"

	return func(auth model.Authorization, t txn.GetFilters) ([]object.Filter, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return []object.Filter{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return []object.Filter{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load this User's hashtag Rules
		criteria := exp.Equal("userId", auth.UserID).AndEqual("type", model.RuleTypeTag)
		rules, err := factory.Rule().Query(session, criteria)

		if err != nil {
			return []object.Filter{}, derp.Wrap(err, location, "Loading hashtag rules")
		}

		// Describe each Rule as a Filter
		result := make([]object.Filter, 0, len(rules))

		for _, rule := range rules {
			result = append(result, filterFromRule(rule))
		}

		// Filters, assemble
		return result, nil
	}
}

// GetFilter implements the Mastodon "get filter" endpoint, returning one of the User's hashtag Rules as a Filter
func GetFilter(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter) (object.Filter, error) {

	const location = "handler.mastodon.GetFilter"

	return func(auth model.Authorization, t txn.GetFilter) (object.Filter, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the hashtag Rule this Filter ID names
		rule, err := loadFilterRule(factory.Rule(), session, auth, t.ID)

		if err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Loading hashtag rule")
		}

		// Filter delivered, no pickles
		return filterFromRule(rule), nil
	}
}

// loadFilterRule loads the User's hashtag Rule that a Filter ID names, or returns a 404 for anything else
func loadFilterRule(ruleService *service.Rule, session data.Session, auth model.Authorization, filterID string) (model.Rule, error) {

	const location = "handler.mastodon.loadFilterRule"

	// Filter IDs are Rule IDs
	ruleID, err := primitive.ObjectIDFromHex(filterID)

	if err != nil {
		return model.Rule{}, errNoSuch(location, "Filter")
	}

	// Load the Rule, which must belong to this User
	rule := model.NewRule()

	if err := ruleService.LoadByID(session, auth.UserID, ruleID, &rule); err != nil {
		return model.Rule{}, derp.Wrap(err, location, "Loading Rule")
	}

	// RULE: Only hashtag Rules are Filters
	if rule.Type != model.RuleTypeTag {
		return model.Rule{}, errNoSuch(location, "Filter")
	}

	// Rule found. Rule returned. Rule-abiding.
	return rule, nil
}

// applyFilter copies a Mastodon filter's title or keyword, action, and expiry onto a hashtag Rule
func applyFilter(rule *model.Rule, title string, keywords []string, filterAction string, expiresIn int, now int64) error {

	// Contexts are ignored on purpose: Rules apply everywhere

	// The hashtag comes from a keyword if one is given, else from the title
	text := title

	for _, keyword := range keywords {
		if strings.TrimSpace(keyword) != "" {
			text = keyword
			break
		}
	}

	// An update may leave the hashtag as it is
	if strings.TrimSpace(text) != "" || rule.Trigger == "" {

		hashtag, err := filterHashtag(text)

		if err != nil {
			return err
		}

		rule.Trigger = hashtag
	}

	// "hide" mutes; "warn" and "blur" label. An empty action leaves it unchanged.
	switch filterAction {
	case "hide":
		rule.Action = model.RuleActionMute
	case "warn", "blur":
		rule.Action = model.RuleActionLabel
	case "":
	default:
		return derp.Validation("filter_action must be hide, warn, or blur")
	}

	// RULE: A LABEL Rule needs label text, which is what apps show on the warning
	if rule.Label == "" {
		rule.Label = rule.Trigger
	}

	// No expires_in means the filter never expires
	rule.ExpireDate = 0

	if expiresIn > 0 {
		rule.ExpireDate = now + int64(expiresIn)
	}

	// Filtered, not stirred
	return nil
}

// filterHashtag reads one hashtag, with or without its "#", from a filter's title or keyword
func filterHashtag(text string) (string, error) {

	// Trim the spaces and any leading "#"
	tag := strings.TrimPrefix(strings.TrimSpace(text), "#")

	// RULE: A filter names exactly one hashtag
	if tag == "" || strings.ContainsAny(tag, " \t\n#") {
		return "", derp.Validation("Filters on this server match one hashtag, such as #politics")
	}

	// Hashtag, you're it
	return "#" + tag, nil
}

// PostFilter implements the Mastodon "create filter" endpoint, saving a hashtag Rule for the User
func PostFilter(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter) (object.Filter, error) {

	const location = "handler.mastodon.PostFilter"

	return func(auth model.Authorization, t txn.PostFilter) (object.Filter, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Start a new hashtag Rule for this User
		// (if one already exists for this tag, Save updates that one instead)
		rule := model.NewRule()
		rule.UserID = auth.UserID
		rule.Type = model.RuleTypeTag

		// Collect the keywords the app sent
		keywords := make([]string, 0, len(t.KeywordsAttributes))
		for _, keyword := range t.KeywordsAttributes {
			keywords = append(keywords, keyword.Keyword)
		}

		// Copy the filter's settings onto the Rule
		if err := applyFilter(&rule, t.Title, keywords, t.FilterAction, int(t.ExpiresIn), time.Now().Unix()); err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Reading filter")
		}

		// Save the Rule
		if err := factory.Rule().Save(session, &rule, "Created by Mastodon API"); err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Saving hashtag rule")
		}

		// It's alive!
		return filterFromRule(rule), nil
	}
}

// PutFilter implements the Mastodon "update filter" endpoint, changing one of the User's hashtag Rules
func PutFilter(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter) (object.Filter, error) {

	const location = "handler.mastodon.PutFilter"

	return func(auth model.Authorization, t txn.PutFilter) (object.Filter, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the hashtag Rule being edited
		rule, err := loadFilterRule(factory.Rule(), session, auth, t.ID)

		if err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Loading hashtag rule")
		}

		// Collect the keywords the app sent, skipping any being removed
		keywords := make([]string, 0, len(t.KeywordsAttributes))
		for _, keyword := range t.KeywordsAttributes {
			if !keyword.Destroy {
				keywords = append(keywords, keyword.Keyword)
			}
		}

		// Copy the filter's settings onto the Rule, remembering its old match key
		previousMatchKey := rule.MatchKey

		if err := applyFilter(&rule, t.Title, keywords, t.FilterAction, int(t.ExpiresIn), time.Now().Unix()); err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Reading filter")
		}

		// RULE: Renaming onto a hashtag that already has a Rule would merge into it and leave this one behind
		if model.RuleMatchKey(rule.Type, rule.Trigger) != previousMatchKey {

			existing := model.NewRule()
			err := factory.Rule().LoadByMatchKey(session, auth.UserID, model.RuleTypeTag, rule.Trigger, &existing)

			if err == nil {
				return object.Filter{}, derp.Validation("You already have a filter for " + rule.Trigger)
			}

			if !derp.IsNotFound(err) {
				return object.Filter{}, derp.Wrap(err, location, "Checking for an existing hashtag rule")
			}
		}

		// Save the Rule
		if err := factory.Rule().Save(session, &rule, "Updated by Mastodon API"); err != nil {
			return object.Filter{}, derp.Wrap(err, location, "Saving hashtag rule")
		}

		// New filter, who dis?
		return filterFromRule(rule), nil
	}
}

// DeleteFilter implements the Mastodon "delete filter" endpoint, removing one of the User's hashtag Rules
func DeleteFilter(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter) (struct{}, error) {

	const location = "handler.mastodon.DeleteFilter"

	return func(auth model.Authorization, t txn.DeleteFilter) (struct{}, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the hashtag Rule being deleted
		rule, err := loadFilterRule(factory.Rule(), session, auth, t.ID)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Loading hashtag rule")
		}

		// Delete the Rule
		if err := factory.Rule().Delete(session, &rule, "Deleted by Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Deleting hashtag rule")
		}

		// Unfiltered, at last
		return struct{}{}, nil
	}
}

// GetFilter_Keywords implements the Mastodon "get filter keywords" endpoint, and always returns an empty list
func GetFilter_Keywords(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Keywords) ([]string, error) {

	return func(model.Authorization, txn.GetFilter_Keywords) ([]string, error) {
		return []string{}, nil
	}
}

// PostFilter_Keyword is the Mastodon "add filter keyword" endpoint, which Emissary does not implement
func PostFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.PostFilter_Keyword) (struct{}, error) {
		// Each filter matches only the hashtag in its title, so there's nowhere to put another keyword
		return struct{}{}, errNotSupported("handler.mastodon.PostFilter_Keyword", "Extra filter keywords")
	}
}

// GetFilter_Keyword implements the Mastodon "get filter keyword" endpoint, and always returns an empty keyword
func GetFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Keyword) (object.FilterKeyword, error) {

	return func(model.Authorization, txn.GetFilter_Keyword) (object.FilterKeyword, error) {
		return object.FilterKeyword{}, nil
	}
}

// PutFilter_Keyword is the Mastodon "update filter keyword" endpoint, which Emissary does not implement
func PutFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter_Keyword) (object.FilterKeyword, error) {

	return func(model.Authorization, txn.PutFilter_Keyword) (object.FilterKeyword, error) {
		return object.FilterKeyword{}, errNoSuch("handler.mastodon.PutFilter_Keyword", "Filter keyword")
	}
}

// DeleteFilter_Keyword is the Mastodon "delete filter keyword" endpoint, which Emissary does not implement
func DeleteFilter_Keyword(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_Keyword) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_Keyword) (struct{}, error) {
		return struct{}{}, errNoSuch("handler.mastodon.DeleteFilter_Keyword", "Filter keyword")
	}
}

// GetFilter_Statuses implements the Mastodon "get filter statuses" endpoint, and always returns an empty list
func GetFilter_Statuses(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Statuses) ([]object.FilterStatus, error) {

	return func(model.Authorization, txn.GetFilter_Statuses) ([]object.FilterStatus, error) {
		return []object.FilterStatus{}, nil
	}
}

// PostFilter_Status is the Mastodon "add filter status" endpoint, which Emissary does not implement
func PostFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_Status) (object.FilterStatus, error) {

	return func(model.Authorization, txn.PostFilter_Status) (object.FilterStatus, error) {
		return object.FilterStatus{}, errNotSupported("handler.mastodon.PostFilter_Status", "Filters")
	}
}

// GetFilter_Status implements the Mastodon "get filter status" endpoint, and always returns an empty status
func GetFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_Status) (object.FilterStatus, error) {

	return func(model.Authorization, txn.GetFilter_Status) (object.FilterStatus, error) {
		return object.FilterStatus{}, nil
	}
}

// DeleteFilter_Status is the Mastodon "delete filter status" endpoint, which Emissary does not implement
func DeleteFilter_Status(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_Status) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_Status) (struct{}, error) {
		return struct{}{}, errNoSuch("handler.mastodon.DeleteFilter_Status", "Filter status")
	}
}

// GetFilters_V1 implements the deprecated V1 "get filters" endpoint, and always returns an empty list
func GetFilters_V1(serverFactory *server.Factory) func(model.Authorization, txn.GetFilters_V1) ([]object.Filter, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetFilters_V1) ([]object.Filter, toot.PageInfo, error) {
		return []object.Filter{}, toot.PageInfo{}, nil
	}
}

// GetFilter_V1 implements the deprecated V1 "get filter" endpoint, and always returns an empty Filter
func GetFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.GetFilter_V1) (object.Filter, error) {
		return object.Filter{}, nil
	}
}

// PostFilter_V1 is the deprecated V1 "create filter" endpoint, which Emissary does not implement
func PostFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.PostFilter_V1) (object.Filter, error) {
		return object.Filter{}, errNotSupported("handler.mastodon.PostFilter_V1", "Filters")
	}
}

// PutFilter_V1 is the deprecated V1 "update filter" endpoint, which Emissary does not implement
func PutFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter_V1) (object.Filter, error) {

	return func(model.Authorization, txn.PutFilter_V1) (object.Filter, error) {
		return object.Filter{}, errNotSupported("handler.mastodon.PutFilter_V1", "Filters")
	}
}

// DeleteFilter_V1 is the deprecated V1 "delete filter" endpoint, which Emissary does not implement
func DeleteFilter_V1(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter_V1) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter_V1) (struct{}, error) {
		return struct{}{}, errNoSuch("handler.mastodon.DeleteFilter_V1", "Filter")
	}
}

// filterFromRule describes an emissary hashtag Rule as a Mastodon Filter
func filterFromRule(rule model.Rule) object.Filter {

	// A trigger may be saved with or without its "#"
	hashtag := "#" + strings.TrimPrefix(rule.Trigger, "#")

	// Hashtag Rules apply everywhere, and match the whole tag
	result := object.Filter{
		ID:           rule.RuleID.Hex(),
		Title:        hashtag,
		Context:      labelFilterContexts,
		FilterAction: "hide",
		Keywords: object.NonNullSlice[object.FilterKeyword]{{
			ID:        rule.RuleID.Hex(),
			Keyword:   hashtag,
			WholeWord: true,
		}},
	}

	// Labels show a warning instead of hiding the post
	if rule.Action == model.RuleActionLabel {
		result.FilterAction = "warn"
	}

	// Rules with an expiry date report it; others never expire
	if rule.ExpireDate > 0 {
		expiresAt := time.Unix(rule.ExpireDate, 0).UTC().Format(time.RFC3339)
		result.ExpiresAt = &expiresAt
	}

	// Spoiler alert: it's a hashtag
	return result
}
