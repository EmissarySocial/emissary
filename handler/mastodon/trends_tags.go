package mastodon

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/toot/object"
	"github.com/benpate/uri"
	"github.com/labstack/echo/v4"
)

// Trending tag list sizes, as Mastodon uses them
const (
	trendingTagsDefault = 10
	trendingTagsMax     = 20
)

// GetTrends_Tags answers Mastodon's trending hashtags with the tags this server's admins have featured
// https://docs.joinmastodon.org/methods/trends/#tags
func GetTrends_Tags(serverFactory *server.Factory) echo.HandlerFunc {

	// Emissary measures no trends, so the list is curated, not computed. It is not in toot.

	const location = "handler.mastodon.GetTrends_Tags"

	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByHostname(serverFactory.Hostname(ctx.Request()))

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		limit := trendingTagsLimit(ctx.QueryParam("limit"))
		criteria := exp.Equal("stateId", model.SearchTagStateFeatured)

		searchTags, err := factory.SearchTag().Query(session, criteria, option.SortAsc("rank"), option.MaxRows(int64(limit)))

		if err != nil {
			return derp.Wrap(err, location, "Querying featured tags")
		}

		host := factory.Hostname()
		result := make([]object.Tag, 0, len(searchTags))

		for _, searchTag := range searchTags {
			if tag, ok := searchTagToTag(searchTag, host); ok {
				result = append(result, tag)
			}
		}

		return ctx.JSON(http.StatusOK, result)
	}
}

// trendingTagsLimit reads the "limit" parameter, keeping it within Mastodon's bounds
func trendingTagsLimit(value string) int {

	limit, err := strconv.Atoi(value)

	if err != nil || limit <= 0 {
		return trendingTagsDefault
	}

	return min(limit, trendingTagsMax)
}

// searchTagToTag describes a featured SearchTag as a Mastodon hashtag named by its normalized Value
func searchTagToTag(searchTag model.SearchTag, host string) (object.Tag, bool) {

	// A display Name may hold spaces, which a hashtag cannot

	if searchTag.Value == "" {
		return object.Tag{}, false
	}

	return object.Tag{
		Name:    searchTag.Value,
		URL:     uri.GuessProtocolForHostname(host) + host + "/tags/" + url.PathEscape(searchTag.Value),
		History: []object.TagHistory{},
	}, true
}
