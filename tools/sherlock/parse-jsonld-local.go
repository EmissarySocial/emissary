package sherlock

import "bytes"

func parseLocalJSONLD(body *bytes.Buffer, data *Page) {
	// TODO: LOW: Add support for JSON-LD metadata embedded in a <script> tag
	// This may be a way to extract the JSON-LD metadata
	// https://pkg.go.dev/github.com/daetal-us/getld#section-readme
}
