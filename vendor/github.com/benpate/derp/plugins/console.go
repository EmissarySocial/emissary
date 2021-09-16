package plugins

import (
	"encoding/json"
	"fmt"
)

// Console prints errors to the system console.
type Console struct{}

// Report implements the `derp.Plugin` interface, which allows the Console
// to be called by the derp.Report() method.
func (console Console) Report(err error) {

	json, _ := json.MarshalIndent(err, "", "\t")
	fmt.Print(string(json))
	return
}
