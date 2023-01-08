package gofed

import (
	"context"
	"net/url"

	"github.com/go-fed/activity/pub"
)

/*
Returns a new pub.Transport for federating with peer software. There is a pub.HttpSigTransport implementation provided for using HTTP and HTTP Signatures, but providing a different transport allows federating using different protocols.

The actorBoxIRI will be either the Inbox or Outbox of an Actor who is attempting to do the dereferencing or delivery. Any authentication scheme applied on the request must be based on this actor. The request must contain some sort of credential of the user, such as a HTTP Signature.

The gofedAgent passed in should be used by the pub.Transport implementation in the User-Agent, as well as the application-specific user agent string. The gofedAgent will indicate this library's use as well as the library's version number.

Any server-wide rate-limiting that needs to occur should happen in a pub.Transport implementation. This factory function allows this to be created, so peer servers are not DOS'd.

Any retry logic should also be handled by the pub.Transport implementation.

Note that the library will not maintain a long-lived pointer to the returned pub.Transport so that any private credentials are able to be garbage collected.

For more information, see the Transports section at https://go-fed.org/ref/activity/pub#Transports
*/

func (common *Common) NewTransport(c context.Context, actorBoxIRI *url.URL, gofedAgent string) (t pub.Transport, err error) {
	return nil, nil
}
