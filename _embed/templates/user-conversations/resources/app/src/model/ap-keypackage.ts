export interface APKeyPackage {
	id: string
	type: "KeyPackage"
	attributedTo: string
	mediaType: "message/mls"
	encoding: "base64"
	content: string
	generator?: {
		id?: "string"
		type?: "Application"
		name?: string
	}
}
