// Package consumer runs Emissary's background work.
//
// The service layer publishes named tasks to the Turbine queue; the Consumer here picks
// them up and dispatches each one, by name, to the function that performs it.  Delivering
// ActivityPub activities, crawling reply trees, moving a User to a new server, connecting
// push services, and importing a profile all run here rather than on the request path.
//
// Every handler takes the same shape -- a Factory, a data Session, and the task's arguments
// -- and returns a queue.Result that tells Turbine whether to retry.  That distinction
// matters: queue.Error asks for a retry, while queue.Failure retires a task that will never
// succeed, so a malformed payload does not cycle forever.
//
// See README.md for the task catalog, and tools/postcommit for why in-transaction callers
// must never publish to the queue directly.
package consumer
