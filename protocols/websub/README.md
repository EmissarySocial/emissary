Right now, WebSub support is scattered throughout the code.  It should eventually be centralized here, or even published as a separate package.

Check out:

/handler/websub.go - Incoming messages about updates on other servers.
/render/step_WebSub.go - WebSub connection negotiation for content on this server.
/service/following_webSub.go - Connecting to external WebSub services.