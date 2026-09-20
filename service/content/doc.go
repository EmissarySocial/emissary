// Package content reads content from sources outside of Emissary, so that it can be copied into
// the Content of a Stream.
//
// Each kind of source gets an Adapter here.  HTTPS is the only one today: it reads one Markdown
// file from a public URL, checks the media type the server declares, and splits any YAML front
// matter off the top.  Adapters return raw bytes and never HTML, because rendering belongs to
// service.Content, which is the only path that sanitizes.
//
// A source address is supplied by a Stream author, so every request passes through Emissary's
// SSRF-guarded HTTP client, and every body is capped before it is read.  See AGENTS.md before
// changing how a source is read.
package content
