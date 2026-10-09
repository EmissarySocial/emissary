// Package replace performs text substitutions inside HTML without breaking the markup.
//
// Content replaces occurrences of a string in the text of an HTML document while skipping
// anything that is not visible text: tag names, attribute values, and the bodies of anchors
// are all left alone.  Linkify builds on the same scanner.
//
// The naive approach -- strings.ReplaceAll over the whole document -- corrupts markup as soon
// as the search term appears in a URL or a class name, and turns a nested link into invalid
// HTML.  This package walks a small state machine instead, so it knows which region it is in.
package replace
