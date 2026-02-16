// APMLSMMessage represents an ActivityPub message that contains
// MLS-encoded content.  This is sent to the ActivityPub outbox, and
// received from the ActivityPub actors mls:messages collection.
export type APMLSMessage = {
	"@context": string
	id: string
	type: string
	mediaType: "message/mls"
	encoding: "base64"
	content: string
	published: string
}
