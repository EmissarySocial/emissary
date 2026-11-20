package consumer

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// fetchLinkPreviewTimeout bounds one page read, the oEmbed hop included
const fetchLinkPreviewTimeout = 30 * time.Second

// FetchLinkPreview reads one linked page's metadata and caches it for preview cards
func FetchLinkPreview(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.FetchLinkPreview"

	link := args.GetString("url")

	// RULE: A task with no usable link can never succeed
	if !service.IsLinkPreviewURL(link) {
		return queue.Failure(derp.BadRequest(location, "Link must be an absolute http or https address", link))
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchLinkPreviewTimeout)
	defer cancel()

	linkPreviewService := factory.LinkPreview()

	// Another server, or an earlier task, may already have read this page
	if _, err := linkPreviewService.LoadByURL(ctx, link); err == nil {
		return queue.Success()
	} else if !derp.IsNotFound(err) {
		return queue.Error(derp.Wrap(err, location, "Checking cached preview", link))
	}

	// A page that cannot be read is cached as a failure, so this ends successfully either way
	preview := linkPreviewService.Fetch(ctx, link, time.Now())
	log.Debug().Str("url", link).Str("status", preview.Status).Msg("Link preview fetched")

	if err := linkPreviewService.Save(ctx, preview); err != nil {
		return queue.Error(derp.Wrap(err, location, "Saving preview", link))
	}

	// Elementary, my dear Watson.
	return queue.Success()
}
