package mastodon

import (
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/filters/
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

		return result, nil
	}
}

// GetFilter implements the Mastodon "get filter" endpoint, and always returns an empty Filter
func GetFilter(serverFactory *server.Factory) func(model.Authorization, txn.GetFilter) (object.Filter, error) {

	return func(model.Authorization, txn.GetFilter) (object.Filter, error) {
		return object.Filter{}, nil
	}
}

// PostFilter is the Mastodon "create filter" endpoint, which Emissary does not implement
func PostFilter(serverFactory *server.Factory) func(model.Authorization, txn.PostFilter) (object.Filter, error) {

	return func(model.Authorization, txn.PostFilter) (object.Filter, error) {
		return object.Filter{}, errNotSupported("handler.mastodon.PostFilter", "Filters")
	}
}

// PutFilter is the Mastodon "update filter" endpoint, which Emissary does not implement
func PutFilter(serverFactory *server.Factory) func(model.Authorization, txn.PutFilter) (object.Filter, error) {

	return func(model.Authorization, txn.PutFilter) (object.Filter, error) {
		return object.Filter{}, errNotSupported("handler.mastodon.PutFilter", "Filters")
	}
}

// DeleteFilter is the Mastodon "delete filter" endpoint, which Emissary does not implement
func DeleteFilter(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFilter) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFilter) (struct{}, error) {
		return struct{}{}, errNoSuch("handler.mastodon.DeleteFilter", "Filter")
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
		return struct{}{}, errNotSupported("handler.mastodon.PostFilter_Keyword", "Filters")
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
	result := object.Filter{
		ID:           rule.RuleID.Hex(),
		Title:        "#" + strings.TrimPrefix(rule.Trigger, "#"),
		FilterAction: "hide",
	}

	// Labels show a warning instead of hiding the post
	if rule.Action == model.RuleActionLabel {
		result.FilterAction = "warn"
	}

	// Spoiler alert: it's a hashtag
	return result
}
