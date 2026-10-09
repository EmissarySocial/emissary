package build

import (
	"bytes"
	"net/http"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/uri"
)

// Navigation -- sending a visitor from this page to another URL -- has two mechanisms, and
// they are NOT interchangeable, because they replace different things.
//
// An HTTP redirect ("Location" plus a 3xx) is performed by whatever transport made the
// request. A browser following a link navigates the whole document, but htmx's XHR follows
// the redirect invisibly and swaps the response into the current page as a fragment.
//
// The "Hx-Redirect" response header is performed by htmx itself, which assigns
// location.href, so it ALWAYS navigates the whole document. It means nothing to a browser
// following a plain <a href>, which would simply land on an empty 200.
//
// The two navigating Steps therefore express two different intents, and each falls back to
// the other's mechanism in the one case where its own cannot work:
//
//	navigateContent   ("redirect-to")   the content lives at another URL
//	navigateDocument  ("forward-to")    the visitor goes somewhere else

// partialRequester is the one fact a navigation decision needs about the caller: whether
// htmx made the request. Every Builder satisfies it, and narrowing the parameter from
// Builder is what lets the decision be exercised without a live database session.
type partialRequester interface {
	IsPartialRequest() bool
}

// navigationURL evaluates a Step's "url" template and confirms that the result is safe to
// send a visitor to.
func navigationURL(urlTemplate *template.Template, builder Builder, location string) (string, error) {

	var target bytes.Buffer

	if err := urlTemplate.Execute(&target, builder); err != nil {
		return "", derp.Wrap(err, location, "Evaluating 'url'")
	}

	// RULE: Reject dangerous or off-site-schemed targets. Step arguments are authored by
	// trusted Template designers, but the values they interpolate are not -- a profile URL
	// or an icon URL arrives from a remote server, and without this guard a "javascript:"
	// scheme (or a protocol-relative "//host" that a browser reads as off-site) becomes the
	// navigation target.
	if uri.NotSafeRedirectURL(target.String()) {
		return "", derp.BadRequest(location, "Unsafe navigation target", target.String())
	}

	return target.String(), nil
}

// navigateContent returns the PipelineBehavior for "the content lives at another URL": an
// HTTP redirect, which lets an htmx caller re-fetch the fragment in place rather than
// reloading the page around it.
func navigateContent(caller partialRequester, statusCode int, target string) PipelineBehavior {

	// RULE: A fragment swap across origins is impossible, not merely undesirable. htmx's XHR
	// would follow the redirect itself, and CORS blocks the cross-origin response, so the
	// click silently does nothing at all. Hand this one navigation to htmx instead.
	if caller.IsPartialRequest() && isOffOrigin(target) {
		return Halt().AsFullPage().WithHeader("Hx-Redirect", target)
	}

	return Halt().AsFullPage().WithStatusCode(statusCode).WithHeader("Location", target)
}

// navigateDocument returns the PipelineBehavior for "the visitor goes somewhere else": an
// "Hx-Redirect" header, which moves the whole browser instead of swapping a fragment into a
// page the visitor is already finished with.
func navigateDocument(caller partialRequester, target string) PipelineBehavior {

	if caller.IsPartialRequest() {
		return Continue().WithEvent("closeModal", "true").WithHeader("Hx-Redirect", target)
	}

	// RULE: "Hx-Redirect" is inert outside htmx, so a caller that ignores it lands on a blank
	// 200. An HTTP redirect is the same intent expressed in the transport that caller DOES
	// understand -- 303, because this branch is reached mostly after a POST, and the visitor
	// should arrive at the new page with a GET.
	return Halt().AsFullPage().WithStatusCode(http.StatusSeeOther).WithHeader("Location", target)
}

// isOffOrigin returns TRUE if a navigation target names a host of its own, which means that
// following it leaves this website.
func isOffOrigin(target string) bool {

	// A relative target is same-origin by definition, so only an absolute URL can leave. This
	// deliberately does NOT compare against the request's own origin: an absolute self-URL is
	// rare, and treating one as off-origin costs only a document navigation where a fragment
	// swap would have done, while a WRONG same-origin match -- a dropped port, a scheme
	// mismatch -- produces a click that silently does nothing. uri.Host returns "" for
	// anything that is not an absolute http(s) URL, which is exactly the test, because
	// navigationURL has already rejected every other scheme.
	return uri.Host(target) != ""
}
