/*
Package handler contains all of the HTTP handlers for Emissary.  This maps loosely to
the "Controllers/Gateways/Presenters" concept in the "Clean Architecture" design pattern
(https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html).
Handlers are wired directly to routes by the main server process, and are the only
entry points for web-based access to Emissary.

Each handler's first task is to determine which domain is being requested.  This is
done by calling the server.Factory and passing it the current request context.
*/
package handler
