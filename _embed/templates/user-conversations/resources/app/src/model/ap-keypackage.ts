// KeyPackage is the ActivityPub representation of a KeyPackage

import type {KeyPackage} from "ts-mls/keyPackage.js"

// https://swicg.github.io/activitypub-e2ee/mls#KeyPackage
export interface APKeyPackage {
	id: string
	type: "mls:KeyPackage"
	attributedTo: string
	to: "as:Public"
	mediaType: "message/mls"
	encoding: "base64"
	content: string
	generator: string
}

// NewAPKeyPackage creates a fully initialized KeyPackage object
// using the provided actorID and public KeyPackage.
export function NewAPKeyPackage(generator: string, actorID: string, publicPackage: KeyPackage): APKeyPackage {
	return {
		id: "", // This will be appened by the server
		type: "mls:KeyPackage",
		to: "as:Public",
		attributedTo: actorID,
		mediaType: "message/mls",
		encoding: "base64",
		generator: generator,
		content: btoa(publicPackage.signature.toString()),
	}
}
