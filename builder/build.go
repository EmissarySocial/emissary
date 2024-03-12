/*
Package build contains build objects, which are passed to HTML templates
to generate HTML pages.  Render objects wrap a specific model object,
providing some safety against direct access to protected data.  Render
objects also include additional methods to query related records in the
database.  For example, the "Stream" builder has queries for `Ancestors`,
`Parent`, `Siblings`, and `Children` streams.

This package also contains implementations for all the action steps available
to template designers.
*/
package builder
