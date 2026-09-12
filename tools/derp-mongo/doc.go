/*
Package derpmongo writes Emissary's runtime errors into a MongoDB collection.

It implements the derp.Reporter interface, so that every derp.Report call
reaches this plugin and becomes one ErrorLog record.  Include and exclude
lists filter by HTTP status code, which lets a server record its own defects
without logging every 404 a visitor stumbles into.

Each record carries a signature, which is the stable identity of one defect,
and a status, which tracks the triage decision made about it.  Every
occurrence of a defect shares one signature, and that is what lets the triage
command collapse a noisy log into a short list of distinct problems.

See ../../triage for the command that reads these records.
*/
package derpmongo
