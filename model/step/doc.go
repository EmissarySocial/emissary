// Package step holds the parsed configuration for every pipeline step a Template can use.
//
// Each file here defines one step: a struct carrying that step's arguments, and a NewXxx
// constructor that compiles the raw HJSON map from a Template into it.  Parsing happens once,
// when a Template is loaded, so a malformed step is caught at load time rather than on the
// request that first reaches it.
//
// Nothing here executes.  The matching step_*.go file in /build does that, reading the struct
// this package produced.  The split keeps Template parsing free of the builder, and lets the
// step catalog be validated without a request, a session, or a database.
//
// STEPS.md documents every step, its attributes, and a worked example.
package step
