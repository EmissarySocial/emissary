// Package blocks renders EditorJS block types that the upstream library does not cover.
//
// goeditorjs ships renderers for the common blocks; the types here -- Code, List, Quote, and
// Table -- fill in the rest, each implementing the library's BlockHandler interface so it can
// be registered alongside the built-ins.
//
// Every renderer writes through benpate/html rather than concatenating strings. Code is
// written as text, so it is escaped. List items and quotes are the editor's inline HTML and
// are written as-is, which is safe only because service.Content.Format sanitizes the whole
// document after these renderers run.
package blocks
