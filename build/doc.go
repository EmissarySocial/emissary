// Package build contains the builders that Emissary passes to HTML templates when rendering
// pages, and the pipeline steps that template designers assemble into actions.
//
// A builder wraps a single model object and exposes a safe, template-friendly view of it.  That
// wrapping is the point: a template reaches only the fields and queries a builder chooses to
// publish, so protected data cannot leak into a page by accident, and related records
// (Ancestors, Parent, Siblings, Children on the Stream builder) are reachable without giving
// templates a database.
//
// The other half of the package is the step catalog.  Every action a template can perform is a
// StepXXX type here, executed as a pipeline against a builder.  Steps implement Get and Post so
// that one definition serves both verbs of an action.
//
// Builders and steps are the layer between handler and template.  Rules and persistence stay in
// service; model/step holds the data definition for each step here.
package build
