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

// populateBundles reads and minifies every Bundle in the provided map, from the provided filesystem
func populateBundles(bundles mapof.Object[model.Bundle], filesystem fs.FS) error {

	for bundleID, bundle := range bundles {

		if err := populateBundle(bundleID, &bundle, filesystem); err != nil {
			return derp.Wrap(err, "service.populateBundles", "Populating bundle", bundleID)
		}

		bundles[bundleID] = bundle
	}

	return nil
}

// populateBundle reads a single Bundle's source files, concatenates them, and minifies the result
func populateBundle(bundleID string, bundle *model.Bundle, filesystem fs.FS) error {

	const location = "service.populateBundle"

	var content bytes.Buffer

	// NILCHECK: Bundle cannot be nil
	if bundle == nil {
		return derp.Internal(location, "Bundle cannot be nil. This should never happen.", bundleID)
	}

	// RULE: Default Caching value to public / 1 hour
	if bundle.CacheControl == "" {
		bundle.CacheControl = "public, max-age=3600"
	}

	// Get the sub-directory for this bundle
	subdirectory, err := fs.Sub(filesystem, bundleID)

	if err != nil {
		return derp.Wrap(err, location, "Reading bundle sub-directory", bundleID)
	}

	// Read all files in the sub-directory
	fileList, err := fs.ReadDir(subdirectory, ".")

	if err != nil {
		return derp.Wrap(err, location, "Reading files in bundle sub-directory", bundleID)
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
			fmt.Println("Unable to read bundle file: " + bundleID + ", " + entry.Name())
			continue
		}

		// Copy the file into the content buffer
		if _, err := io.Copy(&content, file); err != nil {
			return derp.Wrap(err, location, "Copying file into bundle", entry.Name())
		}

		// Add a newline between items.  This will likely be removed by the minifier.
		content.WriteByte('\n')
	}

	// Try to minify the bundle contents
	result, err := minifyContent(bundle.ContentType, &content)

	if err != nil {
		return derp.Wrap(err, location, "Minifying bundle", bundleID)
	}

	// Save the minified content in the bundle
	bundle.Content = result
	return nil
}

// minifyContent compresses HTML, CSS, or JavaScript, passing every other content type through unchanged
func minifyContent(contentType string, content *bytes.Buffer) ([]byte, error) {

	const location = "service.minifyContent"

	// NILCHECK: Content cannot be nil
	if content == nil {
		return nil, derp.Internal(location, "Content cannot be nil. This should never happen.")
	}

	switch contentType {

	case "text/html", "text/css", "text/javascript":

		var result bytes.Buffer

		m := minify.New()
		m.AddFunc("text/html", html.Minify)
		m.AddFunc("text/css", css.Minify)
		m.AddFunc("text/javascript", js.Minify)

		if err := m.Minify(contentType, &result, content); err != nil {
			return nil, derp.Wrap(err, location, "Minifying bundle content.", contentType)
		}

		return result.Bytes(), nil
	}

	return content.Bytes(), nil
}
