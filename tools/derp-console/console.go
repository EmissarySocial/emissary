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

	fmt.Println("")
	console.report(err)
}

func (console Console) report(err error) {

	red := color.New(color.FgRed, color.Bold)
	blue := color.New(color.FgBlue, color.Bold)

	wrappedError := errors.Unwrap(err)

	if wrappedError == nil {
		red.Println("ROOT ERROR: ", derp.Message(err))

	} else {
		console.report(wrappedError)
		blue.Println("- WRAPPED BY:", derp.Message(err))
	}

	if code := derp.ErrorCode(err); code != 0 {
		fmt.Print("- CODE:      ")
		fmt.Println(code, "-", strings.TrimSpace(http.StatusText(code)))
	}

	if location := derp.Location(err); location != "" {
		fmt.Print("- LOCATION:  ")
		fmt.Println(location)
	}

	if details := derp.Details(err); len(details) > 0 {
		for _, detail := range details {

			fmt.Print("- DETAIL:    ")

			switch typed := detail.(type) {

			case string:
				fmt.Println(typed)

			default:
				formatted, _ := json.Marshal(detail)
				fmt.Println(string(formatted))
			}
		}
	}

	fmt.Println("")
}
