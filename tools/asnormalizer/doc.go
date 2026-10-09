// Package asnormalizer reshapes inbound ActivityPub documents into a predictable form.
//
// It is a hannibal streams.Client decorator.  ActivityStreams allows the same fact to be
// written many ways -- a value or a single-element array, a bare string or an object with an
// id, a property present or absent -- and every consumer that reads a document raw has to
// re-handle all of it.  This package collapses those variations once, at load time.
//
// Each supported document type has its own normalizer here (Actor, Announce, Attachment,
// Context, Dislike, and so on), so a new type is added by writing one function rather than
// by touching a switch that everything else depends on.
package asnormalizer
