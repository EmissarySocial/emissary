// Package templates builds the function map available to every Emissary HTML template.
//
// FuncMap starts from rosetta's helpers and overrides or adds the ones Emissary needs:
// Markdown conversion, icons, colors, date formatting, and collection helpers.  Everything a
// template designer can call, other than the builder's own methods, is registered here.
//
// Two rules govern this file.  Any helper returning template.HTML, template.CSS, or
// template.HTMLAttr is a trust boundary, because html/template will not escape what it
// returns -- so it must sanitize or escape its own inputs.  And rosetta's funcmap replaces
// the "and" and "or" builtins with strict-boolean versions, which changes what templates may
// write.  AGENTS.md covers both.
package templates
