
// MLSMessage represents an ActivityPub message that contains MLS-encrypted data
// There are five message types, which all map into ActivityStream format.
// https://swicg.github.io/activitypub-e2ee/mls#types
export type APMLSMessage = {
	"@context": any
	id: string
	type: "GroupInfo" | "KeyPackage" | "PrivateMessage" | "PublicMessage" | "Welcome"
	attributedTo: string
	to: string | string[]
	mediaType: "message/mls"
	encoding: "base64"
	summary: string
	content: string
	generator: string
}
