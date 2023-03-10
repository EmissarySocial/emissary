/*
Package model contains all of the domain model objects that are used in the system.
Emissary uses a variation the "CLEAN" architecture design pattern, so model objects
do not contain business logic -- only pure data and limited functionality (such as
validation).  Database access, business logic, and other more advanced features
are handled by the service package.
*/
package model
