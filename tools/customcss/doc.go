// Package customcss checks a stylesheet that one person writes for a page that other people
// view, such as the custom CSS on a profile.  It never rewrites a stylesheet: Validate either
// accepts the author's text exactly as written, or returns a validation error that names the
// rule it breaks and the line it breaks it on.
//
// An accepted stylesheet is safe to place inside a <style> element.  It cannot contain "</", so
// it cannot close the element.  Every block it opens is closed, so a template can wrap it in
// @scope without the stylesheet reaching outside.  And it loads nothing from another website,
// so it cannot report who is reading the page.  The rules run against the tokens of a CSS
// lexer, so a brace or url() inside a string or comment is read the way a browser reads it.
//
// The package registers SchemaFormat with rosetta, so a schema applies the same check when a
// stylesheet is saved, and the template funcmap applies it again whenever one is rendered.
package customcss
