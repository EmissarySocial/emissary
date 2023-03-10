/*
Package service includes all of the services used in Emissary.  This maps loosely
to the "Use Cases" concept in the "Clean Architecture" design pattern
(https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

Services include server-level singletons that are used by every domain (such as themes,
templates, etc) and domain-level services that have unique instances for each domain.

* Server-level serivces are created by the server.Factory and are passed by reference
to each domain factory.

* Domain-level services are created by the domain.Factory, and typically require a
connection to a database table, which is why they are not global.
*/
package service
