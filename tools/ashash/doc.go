// Package ashash resolves ActivityPub URLs that point inside a document.
//
// It is a hannibal streams.Client decorator.  When an id carries a fragment ("#..."), it
// loads the document at the base URL and then searches that document for the named value,
// instead of requesting a URL that no server will answer.
//
// Fragment ids are common in ActivityPub -- a key, a tag, or an embedded object is routinely
// published as a fragment of its parent actor or activity -- and fetching one directly
// returns the parent, or nothing at all.  Ids without a fragment pass straight through.
package ashash
