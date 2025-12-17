
// APKeyPackage is the ActivityPub representation of a KeyPackage

import type { KeyPackage, PrivateKeyPackage } from "ts-mls"

// https://swicg.github.io/activitypub-e2ee/mls#KeyPackage
export type APKeyPackage = {
	// type: ["Object", "KeyPackage"]
	type: "KeyPackage"
	id: string
	attributedTo: string
	to: "as:Public"
	summary: string
	mediaType: "message/mls"
	encoding: "base64"
	content: string
	generator: string
}

// NewAPKeyPackage creates a fully initialized APKeyPackage object
export function NewAPKeyPackage(actorID:string, publicPackage:KeyPackage): APKeyPackage {
	return {
		id:"", // This will be appened by the server
		// type: ["Object", "KeyPackage"],
		type: "KeyPackage",
		to: "as:Public",
		attributedTo: actorID,
		mediaType: "message/mls",
		encoding: "base64",
		summary: "",
		generator: "Emissary MLS",
		content: btoa(publicPackage.signature.toString()),
	}
}

// Record stored in the local IndexedDB database
export type IDBKeyPackage = {
	id: string // The URL of this KeyPackage
	publicPackage: KeyPackage
	privatePackage: PrivateKeyPackage
}