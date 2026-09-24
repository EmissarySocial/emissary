package service

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
)

// SetStatusPollError records a failed Actor load against a Following, marking it GONE when the
// remote server says the account was deleted, and FAILURE (which escalates to PAUSED) otherwise.
func (service *Following) SetStatusPollError(session data.Session, following *model.Following, err error) error {

	const location = "service.Following.SetStatusPollError"

	// RULE: A 429 rate-limits the whole HOST, so callers requeue it and never call this method.
	// Anything that arrives here is a fact about the Following itself.
	statusMessage := followingStatusMessage(err)

	// RULE: A 410 is the remote server stating that the account was DELETED.  Mastodon answers
	// 410 for a deleted account, so GONE needs no waiting period.
	if derp.ErrorCode(err) == http.StatusGone {

		if inner := service.SetStatusGone(session, following, statusMessage); inner != nil {
			return derp.Wrap(inner, location, "Marking Following as gone", following.URL)
		}

		return nil
	}

	// RULE: Everything else -- 4xx, 5xx, DNS, TLS, timeout -- is RECORDED as a failure.  A dead
	// domain never answers 4xx, and recording is what eventually escalates the record to PAUSED.
	if inner := service.SetStatusPollFailure(session, following, statusMessage); inner != nil {
		return derp.Wrap(inner, location, "Marking Following as failed", following.URL)
	}

	// Better luck next time
	return nil
}

// followingStatusMessage renders a failed Actor load as a short sentence for the Following's owner
func followingStatusMessage(err error) string {

	// RULE: A response that arrived is never described as unreachable.  derp reports this as a
	// 500, so without this branch the owner was told "Could not reach this server: 200 OK".
	if status, contentType, answered := answeredWithoutActor(err); answered {

		detail := strconv.Itoa(status)

		if contentType != "" {
			detail += ", " + contentType
		}

		return "This address did not return an account (" + detail + "). It may be a web page or a feed rather than a fediverse account."
	}

	code := derp.ErrorCode(err)

	switch code {

	case http.StatusUnauthorized, http.StatusForbidden:
		return "This account refused our request (" + strconv.Itoa(code) + "). It may be private, or may have blocked this server."

	case http.StatusNotFound:
		return "This account could not be found (" + strconv.Itoa(code) + "). It may have been moved or deleted."

	case http.StatusGone:
		return "This account has been deleted (410)."
	}

	if derp.IsClientError(err) {
		return "Unable to read this account (" + strconv.Itoa(code) + ")."
	}

	// RULE: Everything else may be a real 5xx or a transport failure that never reached a
	// server, and derp reports both as 500 -- so quote the reason instead of naming a code.
	return "Could not reach this server: " + derp.RootMessage(err)
}

// answeredWithoutActor reports whether a failed Actor load was a 2xx response that arrived intact
// but held no Actor, returning the status and media type it arrived as.
func answeredWithoutActor(err error) (int, string, bool) {

	// RULE: The header is written by the remote server, so it is bounded before it is quoted.
	// Every real media type fits easily; this only stops a hostile or broken one filling the message.
	const contentTypeMaxLength = 100

	// The response that arrived is carried by the HTTPError, whatever code the wrapping added
	var httpError derp.HTTPError

	if !errors.As(err, &httpError) {
		return 0, "", false
	}

	// RULE: Only a 2xx means the request completed.  A 4xx is a refusal, a 5xx is the server's
	// own fault, and a transport failure never produced a response to read at all.
	status := httpError.Response.StatusCode

	if (status < http.StatusOK) || (status >= http.StatusMultipleChoices) {
		return 0, "", false
	}

	// Strip the parameters, so a "; charset=utf-8" never reaches the owner's sentence
	contentType, _, _ := strings.Cut(httpError.Response.Header.Get("Content-Type"), ";")
	contentType = strings.TrimSpace(contentType)

	// RULE: A cut can split a multi-byte character, so the broken fragment at the end is dropped
	contentType = strings.ToValidUTF8(contentType[:min(len(contentType), contentTypeMaxLength)], "")

	return status, contentType, true
}
