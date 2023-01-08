## The FederatingProtocol Interface

The pub.FederatingProtocol is only needed if an application wants to do the S2S (Server-to-server, or federating) ActivityPub protocol. It supplements the pub.CommonBehavior interface with the additional methods required by a federating application.

PostInboxRequestBodyHook(c context.Context, r *http.Request, activity Activity) (context.Context, error)

This is a hook that occurs after reading the request body of a POST request to an Actor's Inbox.

Provides your application the opportunity to set contextual information based on the incoming http.Request and its body. Some applications simply return c and do nothing else, which is OK. More commonly, software simply inspects the http.Request path to determine the actual local Actor being interacted with, and save such information within c.

Any errors returned immediately abort processing of the request and are returned to the caller of the Actor's PostInbox.

Do not do anything sensitive in this method. Neither authorization nor authentication has been attempted at the point PostInboxRequestBodyHook has been called.

AuthenticatePostInbox(c context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error)

This is a callback for your application to determine whether the incoming http.Request is authenticated and, implicitly, authorized to proceed with processing the request.

If an error is returned, it is passed back to the caller of PostInbox. In this case, the implementation must not write a response to the http.ResponseWriter as is expected that the client will do so when handling the error. The authenticated value is ignored in this case.

If no error is returned, but your application determines that authentication or authorization fails, then authenticated must be false and err nil. It is expected that the implementation handles writing to the http.ResponseWriter in this case.

Finally, if the authentication and authorization succeeds, then authenticated must be true and err nil. The request will continue to be processed.

Blocked(c context.Context, actorIRIs []*url.URL) (blocked bool, err error)

Given a list of actorIRIs, determines whether any are blocked for this particular request context and based on the particular application's state. For example, some applications allow users or software instances to maintain lists of blocked peer Actors or domains.

To determine the current user being interacted with, it is recommended to set such information in the PostInboxRequestBodyHook method.

If an error is returned, it is passed back to the caller of PostInbox.

If no error is returned, but the interaction should be blocked, then blocked must be true and err nil. An http.StatusForbidden will be written in the response.

Finally, the interaction should proceed, then blocked must be false and err nil. The request will continue to be processed.

FederatingCallbacks(c context.Context) (wrapped FederatingWrappedCallbacks, other []interface{}, err error)

Returns the application-specific logic needed for your application as callbacks for the library to invoke.

The library splits your applications behaviors between those specified in the ActivityPub spec, which will wrap your behaviors in wrapped, or behaviors not known in the ActivityPub spec which will be provided in other.

The pub.FederatingWrappedCallbacks returned provides a collection of default ActivityPub behaviors as defined in the specification. For more details on how to use these provided behaviors and supplement with your own business logic, see Federating Wrapped Callbacks. The zero-value is a valid value.

If instead you wish to override the default ActivityPub behaviors, such as doing nothing, then the other return value should contain a function with a signature like:

other = []interface{}{
	// This function overrides the FederatingWrappedCallbacks-provided behavior
	func(c context.Context, create vocab.ActivityStreamsCreate) error {
		return nil
	},
}
The above would replace the library's default behavior of creating the entry in the database upon receiving a Create activity.

If you want to handle an Activity that does not have a default behavior provided in pub.FederatingWrappedCallbacks, then specify it in other using a similar function signature.

Applications are not expected to handle every single ActivityStreams type and extension. The unhandled ones are passed to DefaultCallback.

DefaultCallback(c context.Context, activity Activity) error

This method is called for types that the library can deserialize but is not handled by the application's callbacks returned in the FederatingCallbacks method.

MaxInboxForwardingRecursionDepth(c context.Context) int

MaxInboxForwardingRecursionDepth determines how deep to search within an activity's historical chain to determine if inbox forwarding needs to occur. After reaching this depth, it is assumed that peers deeper than that conversational depth are no longer candidates for triggering the inbox forwarding logic.

Zero or negative numbers indicate recurring infinitely, which can result in your application being manipulated by malicious peers. Do not return a value of zero nor a negative number.

MaxDeliveryRecursionDepth(c context.Context) int

This method determines how deep to search within collections owned by peers when they are targeted to receive a delivery. After reaching this depth, it is assumed that peers deeper than that are no longer interested in receiving messages. A positive number must be returned.

Zero or negative numbers indicate recurring infinitely, which can result in your application being manipulated by malicious peers. Do not return a value of zero nor a negative number.

FilterForwarding(c context.Context, potentialRecipients []*url.URL, a Activity) (filteredRecipients []*url.URL, err error)

Allows the implementation to apply outbound message business logic such as blocks, spam filtering, and so on to a list of potentialRecipients when inbox forwarding has been triggered. Your application must apply some sort of filtering, such as limiting delivery to an actor's followers. Otherwise, your application will become a vector for spam on behalf of malicious peers, and users of your software will be mass-blocked by their peers.

The activity is provided as a reference for more intelligent logic to be used, but the implementation must not modify the activity.

GetInbox(c context.Context, r *http.Request) (vocab.ActivityStreamsOrderedCollectionPage, error)

Returns a proper paginated view of the Inbox for serving in a response. Since AuthenticateGetInbox is called before this, the implementation is responsible for ensuring things like proper pagination, visible content based on permissions, and whether to leverage the pub.Database's GetInbox method in this implementation.