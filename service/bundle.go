package service

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
)

func populateBundles(bundles mapof.Object[model.Bundle], filesystem fs.FS) error {

	for bundleID, bundle := range bundles {

		if err := populateBundle(bundleID, &bundle, filesystem); err != nil {
			return derp.Wrap(err, "service.populateBundles", "Error populating bundle", bundleID)
		}

		bundles[bundleID] = bundle
	}

	return nil
}

func populateBundle(bundleID string, bundle *model.Bundle, filesystem fs.FS) error {

	var content bytes.Buffer

	// RULE: Default Caching value to public / 1 hour
	if bundle.CacheControl == "" {
		bundle.CacheControl = "public, max-age=3600"
	}

	// Get the sub-directory for this bundle
	subdirectory, err := fs.Sub(filesystem, bundleID)

	if err != nil {
		return derp.Wrap(err, "service.populateBundle", "Error reading bundle directory", bundleID)
	}

	// Read all files in the sub-directory
	fileList, err := fs.ReadDir(subdirectory, ".")

	if err != nil {
		return derp.Wrap(err, "service.populateBundle", "Error reading bundle directory", bundleID)
	}

	// Import every file into the bundle
	for _, entry := range fileList {

		// Skip sub-directories
		if entry.IsDir() {
			continue
		}

		// Open the file
		file, err := subdirectory.Open(entry.Name())

		if err != nil {
			fmt.Println("Error reading bundle file: " + bundleID + ", " + entry.Name())
			continue
		}

		// Copy the file into the content buffer
		if _, err := io.Copy(&content, file); err != nil {
			return derp.Wrap(err, "service.populateBundle", "Error reading file", entry.Name())
		}

		// Add a newline between items.  This will likely be removed by the minifier.
		content.WriteByte('\n')
	}

	// Try to minify the bundle contents
	result, err := minifyContent(bundle.ContentType, &content)

	if err != nil {
		return derp.Wrap(err, "service.populateBundle", "Error minifying bundle", bundleID)
	}

	// Save the minified content in the bundle
	bundle.Content = result
	return nil
}

func minifyContent(contentType string, content *bytes.Buffer) ([]byte, error) {

	switch contentType {

	case "text/html", "text/css", "text/javascript":

		var result bytes.Buffer

		m := minify.New()
		m.AddFunc("text/html", html.Minify)
		m.AddFunc("text/css", css.Minify)
		m.AddFunc("text/javascript", js.Minify)

		if err := m.Minify(contentType, &result, content); err != nil {
			return nil, derp.Wrap(err, "service.minifyContent", "Error minifying HTML")
		}

		return result.Bytes(), nil
	}

	return content.Bytes(), nil
}
