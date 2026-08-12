// Package model defines Emissary's domain data structures -- the objects that are stored in the
// database and passed through the application: Stream, User, Follower, Following, Rule,
// Conversation, KeyPackage, and many more.
//
// Models carry their own data and the vocabulary that describes it: field constants, state and
// type enumerations, JSON-schema definitions, and the accessors that let a record be read and
// written generically.  They do not carry business logic.  Loading, validating, and the side
// effects of a change all belong to the service layer, which is what keeps a model safe to
// construct and inspect anywhere.
//
// Each model typically spans a few files by role -- the struct and its constructor, an
// _accessors file implementing the schema and getter/setter plumbing, and an _activitypub or
// _jsonld file for its federated representation.
//
// The step subdirectory holds the data definitions for each builder pipeline step; the building
// functions themselves live in the build package.
package model
