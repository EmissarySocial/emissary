/*
Package handler contains all of the HTTP handlers for Emissary.
These are wired directly into the echo server, and are the only
entry points for all web-based access.

Each handler's first task is to determine which domain is being
requested.  This is done by calling the server.Factory and
passing it the current request context.
*/
package handler
