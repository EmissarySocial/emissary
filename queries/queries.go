// Package queries contains all of the custom queries required by this application
// that DO NOT run through the standard `data` package.  These are queries that rely
// on specific features of the database (such as mongodb aggregation, or live queries)
// that are out of scope for the data package
//
// If this application is ever migrated from mongodb, these functions will need to
// be rewritten to match the new database API.
//
// This package is an abberation in the "Clean Architecture" design pattern
// (https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html),
// but it is useful for now in order to maintain some flexibility in the database.
package queries
