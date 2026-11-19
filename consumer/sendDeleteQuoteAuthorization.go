package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// SendDeleteQuoteAuthorization revokes a quote (FEP-044f).  It loads the quote post to learn who
// quoted, then queues a Delete of the QuoteAuthorization for that actor and the author's followers.
func SendDeleteQuoteAuthorization(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SendDeleteQuoteAuthorization"

	// RULE: The task must name the quote post, the QuoteAuthorization, and the actor who signs
	quoteURL := args.GetString("quote")

	if (quoteURL == "") || (args.GetString("authorization") == "") || (args.GetString("actor") == "") {
		return queue.Failure(derp.Internal(location, "Task requires 'quote', 'authorization', and 'actor'", args))
	}

	// Find the actor who quoted, from the quote post itself
	quoterURL, err := quoteAuthor(factory.ActivityStream().AppClient(), quoteURL)

	// RULE: A lookup that may yet succeed is tried again, before anything is sent
	if result, retry := retryQuoteLookup(err); retry {
		return result
	}

	// Queue the Delete for delivery
	factory.Outbox().DeliverDeleteQuoteAuthorization(session, args, quoterURL)

	// Return to sender.
	return queue.Success()
}

// retryQuoteLookup decides whether a failed load of the quote post is worth another attempt.  It
// returns the Result that retries the task, and TRUE, or FALSE when the task should go on.
func retryQuoteLookup(err error) (queue.Result, bool) {

	const location = "consumer.retryQuoteLookup"

	// A quote post that loaded needs no retry
	if err == nil {
		return queue.Success(), false
	}

	// RULE: Check for 429 first, because it is a client error too.  A throttled host is asked again when it says to
	if isTooMany, delay := derp.IsTooManyRequests(err); isTooMany {
		return queue.Requeue(delay), true
	}

	// A quote post that is gone, or closed to us, names nobody.  The followers are still told
	if derp.IsClientError(err) {
		log.Debug().Err(err).Msg("SendDeleteQuoteAuthorization: quote post cannot be read; telling followers only")
		return queue.Success(), false
	}

	// Anything else may pass, as when the quoting server is down, so try again later
	return queue.Error(derp.Wrap(err, location, "Loading quote post")), true
}

// quoteAuthor returns the ID of the actor who wrote the quote post at the provided URL.
// It returns an empty string, and no error, when the post names no author.
func quoteAuthor(client streams.Client, quoteURL string) (string, error) {

	const location = "consumer.quoteAuthor"

	quote, err := client.Load(quoteURL)

	if err != nil {
		return "", derp.Wrap(err, location, "Loading quote post", quoteURL)
	}

	return quote.AttributedTo().ID(), nil
}
