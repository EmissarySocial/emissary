// Package blocks renders EditorJS block types that the upstream library does not cover.
//
// goeditorjs ships renderers for the common blocks; the types here -- Code, List, Quote, and
// Table -- fill in the rest, each implementing the library's BlockHandler interface so it can
// be registered alongside the built-ins.
//
// Every renderer writes through benpate/html rather than concatenating strings, so block
// content coming out of a browser editor is escaped on the way into the page.
package blocks
