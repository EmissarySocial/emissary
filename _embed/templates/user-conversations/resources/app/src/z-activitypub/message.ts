
// PlaintextMessage represents a simple ActivityPub Note without any encryption
export type APMessage = {
	id: string
	type:"Note" | "Article"
	attributedTo: string
	to: string[]
	context: string
	inReplyTo?:string
	content: string
	published: string
}