## The SocialProtocol Interface

The pub.SocialProtocol is only needed if an application wants to do the C2S (Client-to-server, or social) ActivityPub protocol. It supplements the pub.CommonBehavior interface with the additional methods required by a social application.

PostOutboxRequestBodyHook(c context.Context, r *http.Request, data vocab.Type) (context.Context, error)

This is a hook that occurs after reading the request body of a POST request to an Actor's Outbox.

Provides your application the opportunity to set contextual information based on the incoming http.Request and its body. Some applications simply return c and do nothing else, which is OK. More commonly, software simply inspects the http.Request path to determine the actual local Actor being interacted with, and save such information within c.

Any errors returned immediately abort processing of the request and are returned to the caller of the Actor's PostOutbox.

Do not do anything sensitive in this method. Neither authorization nor authentication has been attempted at the point PostOutboxRequestBodyHook has been called.

AuthenticatePostOutbox(c context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error)

This is a callback for your application to determine whether the incoming http.Request is authenticated and, implicitly, authorized to proceed with processing the request.

If an error is returned, it is passed back to the caller of PostOutbox. In this case, the implementation must not write a response to the http.ResponseWriter as is expected that the client will do so when handling the error. The authenticated value is ignored in this case.

If no error is returned, but your application determines that authentication or authorization fails, then authenticated must be false and err nil. It is expected that the implementation handles writing to the http.ResponseWriter in this case.

Finally, if the authentication and authorization succeeds, then authenticated must be true and err nil. The request will continue to be processed.

SocialCallbacks(c context.Context) (wrapped SocialWrappedCallbacks, other []interface{}, err error)

Returns the application-specific logic needed for your application as callbacks for the library to invoke.

The library splits your applications behaviors between those specified in the ActivityPub spec, which will wrap your behaviors in wrapped, or behaviors not known in the ActivityPub spec which will be provided in other.

The pub.SocialWrappedCallbacks returned provides a collection of default ActivityPub behaviors as defined in the specification. For more details on how to use these provided behaviors and supplement with your own business logic, see Social Wrapped Callbacks. The zero-value is a valid value.

If instead you wish to override the default ActivityPub behaviors, such as doing nothing, then the other return value should contain a function with a signature like:

other = []interface{}{
	// This function overrides the SocialWrappedCallbacks-provided behavior
	func(c context.Context, create vocab.ActivityStreamsCreate) error {
		return nil
	},
}
The above would replace the library's default behavior of creating the entry in the database upon receiving a Create activity.

If you want to handle an Activity that does not have a default behavior provided in pub.SocialWrappedCallbacks, then specify it in other using a similar function signature.

Applications are not expected to handle every single ActivityStreams type and extension. The unhandled ones are passed to DefaultCallback.

DefaultCallback(c context.Context, activity Activity) error

This method is called for types that the library can deserialize but is not handled by the application's callbacks returned in the SocialCallbacks method.

