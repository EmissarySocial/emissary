// Package nodeinfo reads a fediverse server's NodeInfo document.
//
// NodeInfo is the convention that lets one server describe itself to another: software name
// and version, supported protocols, open registrations, and rough usage counts.  Discovery
// runs in two hops -- fetch /.well-known/nodeinfo, then follow the link it advertises -- and
// the Client here performs both.
//
// Every field is self-reported by the remote server and none of it is verified, so treat a
// NodeInfo as a claim.  Emissary uses it to describe peers, never to make a trust decision.
package nodeinfo
