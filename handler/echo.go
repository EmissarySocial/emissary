package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// echoMaxBodyBytes caps the request body that GetEcho reads back
const echoMaxBodyBytes = 65535

// echoRedacted stands in for every credential that GetEcho would otherwise print
const echoRedacted = "(redacted)"

// GetEcho returns the request as plain text, with its credentials redacted, for the domain owner to debug with
func GetEcho(ctx *steranko.Context, factory *service.Factory, _ data.Session) error {

	const location = "handler.GetEcho"

	request := ctx.Request()
	var buffer bytes.Buffer

	// Write the request line and the addresses the server sees
	fmt.Fprintf(&buffer, "HTTP REQUEST ECHO\n\n")
	fmt.Fprintf(&buffer, "  %-12s %s\n", "Method:", request.Method)
	fmt.Fprintf(&buffer, "  %-12s %s\n", "URL:", request.URL.String())
	fmt.Fprintf(&buffer, "  %-12s %s\n", "Proto:", request.Proto)
	fmt.Fprintf(&buffer, "  %-12s %s\n", "Remote:", ctx.RealIP())
	fmt.Fprintf(&buffer, "  %-12s %s\n", "ClientIP:", factory.ClientIP(request))
	fmt.Fprintf(&buffer, "  %-12s %s\n", "Host:", request.Host)

	// Write the query parameters
	if queryParams := ctx.QueryParams(); len(queryParams) > 0 {
		fmt.Fprintf(&buffer, "\nQuery Parameters\n")
		echoValues(&buffer, queryParams)
	}

	// Write the headers.  RULE: A header that carries a credential is named and never printed
	fmt.Fprintf(&buffer, "\nHeaders\n")
	for _, name := range echoSortedKeys(request.Header) {
		for _, value := range request.Header[name] {
			fmt.Fprintf(&buffer, "  %-30s %s\n", name+":", echoHeaderValue(name, value))
		}
	}

	// Write the cookie names.  RULE: A cookie's value is never printed, because the session is one
	if cookies := request.Cookies(); len(cookies) > 0 {
		fmt.Fprintf(&buffer, "\nCookies\n")
		for _, cookie := range cookies {
			fmt.Fprintf(&buffer, "  %-20s %s\n", cookie.Name+":", echoRedacted)
		}
	}

	// RULE: Read a bounded body, so a large request cannot exhaust memory
	body, err := io.ReadAll(io.LimitReader(request.Body, echoMaxBodyBytes))

	if err != nil {
		return derp.Wrap(err, location, "Reading request body", derp.WithBadRequest())
	}

	defer derp.ReportFunc(request.Body.Close)

	// Write the body
	fmt.Fprintf(&buffer, "\nBody (%d bytes)\n", len(body))
	echoBody(&buffer, request.Header.Get("Content-Type"), body)

	return ctx.String(http.StatusOK, buffer.String())
}

// echoHeaderValue returns the header's value, or a placeholder if the header carries a credential
func echoHeaderValue(name string, value string) string {

	switch http.CanonicalHeaderKey(name) {
	case "Authorization", "Cookie", "Proxy-Authorization":
		return echoRedacted
	}

	return value
}

// echoBody writes a request body to the buffer, as form values where it holds a form
func echoBody(buffer *bytes.Buffer, contentType string, body []byte) {

	if len(body) == 0 {
		fmt.Fprintf(buffer, "  (empty)\n")
		return
	}

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if values, err := url.ParseQuery(string(body)); err == nil {
			echoValues(buffer, values)
			return
		}
	}

	for line := range strings.SplitSeq(string(body), "\n") {
		fmt.Fprintf(buffer, "  %s\n", line)
	}
}

// echoValues writes every value to the buffer, ordered by name
func echoValues(buffer *bytes.Buffer, values map[string][]string) {

	for _, name := range echoSortedKeys(values) {
		for _, value := range values[name] {
			fmt.Fprintf(buffer, "  %-20s %s\n", name+":", value)
		}
	}
}

// echoSortedKeys returns the names in a set of values, in alphabetical order
func echoSortedKeys(values map[string][]string) []string {

	result := make([]string, 0, len(values))

	for name := range values {
		result = append(result, name)
	}

	slices.Sort(result)
	return result
}
