// Package console is a derp.Reporter that reports errors to the console in a pretty format.
package derpconsole

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/benpate/derp"
	"github.com/fatih/color"
)

type Console struct{}

func New() Console {
	return Console{}
}

func (console Console) Report(err error) {
	_, _ = fmt.Println("")
	console.report(err)
}

func (console Console) report(err error) {

	red := color.New(color.FgRed, color.Bold)   // nolint:scopeguard
	blue := color.New(color.FgBlue, color.Bold) // nolint:scopeguard

	if wrappedError := errors.Unwrap(err); wrappedError == nil {
		_, _ = red.Println("ROOT ERROR: ", derp.Message(err))

	} else {
		console.report(wrappedError)
		_, _ = blue.Println("- WRAPPED BY:", derp.Message(err))
	}

	if code := derp.ErrorCode(err); code != 0 {
		_, _ = fmt.Print("- CODE:      ")
		_, _ = fmt.Println(code, "-", strings.TrimSpace(http.StatusText(code)))
	}

	if location := derp.Location(err); location != "" {
		_, _ = fmt.Print("- LOCATION:  ")
		_, _ = fmt.Println(location)
	}

	if details := derp.Details(err); len(details) > 0 {
		for _, detail := range details {

			_, _ = fmt.Print("- DETAIL:    ")

			switch typed := detail.(type) {

			case string:
				_, _ = fmt.Println(typed)

			default:
				formatted, _ := json.Marshal(detail)
				_, _ = fmt.Println(string(formatted))
			}
		}
	}

	fmt.Println("")
}
