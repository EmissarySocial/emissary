// Package console is a derp.Reporter that reports errors to the console in a pretty format.
package derpconsole

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/benpate/derp"
	"github.com/fatih/color"
)

// output is the stream every report is written to.  RULE: errors go to STDERR, matching the
// application's zerolog output (also stderr) so operational logs and error reports share one
// stream — consistent under redirection (e.g. `2>logs`) and correctly ordered for a collector.
// stdout is reserved for a program's actual output data, which a long-running server has none of.
var output = os.Stderr

// Console is a derp.Reporter that writes a human-readable error report to the console
type Console struct{}

// New returns a fully initialized Console reporter
func New() Console {
	return Console{}
}

// Report implements the derp.Plugin interface, printing the full error chain
func (console Console) Report(err error) {
	_, _ = fmt.Fprintln(output, "") //nolint:errcheck
	console.report(err)
}

// report walks the error chain from the root outward, printing one section per link
func (console Console) report(err error) {

	red := color.New(color.FgRed, color.Bold)   // nolint:scopeguard
	blue := color.New(color.FgBlue, color.Bold) // nolint:scopeguard

	if wrappedError := errors.Unwrap(err); wrappedError == nil {
		_, _ = red.Fprintln(output, "ROOT ERROR: ", derp.Message(err))

	} else {
		console.report(wrappedError)
		_, _ = blue.Fprintln(output, "- WRAPPED BY:", derp.Message(err))
	}

	if code := derp.ErrorCode(err); code != 0 {
		_, _ = fmt.Fprint(output, "- CODE:      ")
		_, _ = fmt.Fprintln(output, code, "-", strings.TrimSpace(http.StatusText(code)))
	}

	if location := derp.Location(err); location != "" {
		_, _ = fmt.Fprint(output, "- LOCATION:  ")
		_, _ = fmt.Fprintln(output, location)
	}

	if details := derp.Details(err); len(details) > 0 {
		for _, detail := range details {

			_, _ = fmt.Fprint(output, "- DETAIL:    ")

			switch typed := detail.(type) {

			case string:
				_, _ = fmt.Fprintln(output, typed)

			default:
				formatted, _ := json.Marshal(detail)
				_, _ = fmt.Fprintln(output, string(formatted))
			}
		}
	}

	_, _ = fmt.Fprintln(output, "")
}
