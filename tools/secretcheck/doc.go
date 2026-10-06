// Package secretcheck lets tests prove that an error does not carry a secret.
//
// Errors in this codebase are reported in three forms: the error message, a JSON encoding
// printed by the console reporter, and a BSON encoding stored in the ErrorLog collection by
// the mongo reporter. A secret can escape through any one of them. The BSON form matters
// most, because it follows bson tags, so a field hidden from JSON with `json:"-"` is still
// stored. A test that checks only the message or only the JSON can pass while the secret
// still reaches the database.
//
// Find reports every form that contains a secret, and RequireAbsent fails a test when any
// form does. The BSON form is derpmongo.StoredError's rebuild, so a chain holding a mongo
// driver error is checked whole. This package is for tests, but it lives outside a _test.go file so that every
// package can share one implementation.
package secretcheck
