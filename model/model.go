/*
Package model contains all of the domain model objects that are used in the system. Model objects
map loosely to the concept of "Entities" outlined in the "Clean Architecture" design pattern
(https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html),so model objects
do not contain business logic -- only pure data and limited functionality (such as validation).
Database access, business logic, and other more advanced features are handled by the service package.
*/
package model
