// KeyPackage is the ActivityPub representation of a KeyPackage

import {bytesToBase64} from "ts-mls"
import {decode} from "ts-mls/codec/tlsDecoder.js"
import {encode} from "ts-mls/codec/tlsEncoder.js"
import type {KeyPackage} from "ts-mls/keyPackage.js"
import {mlsMessageEncoder} from "ts-mls/message.js"
import {mlsMessageDecoder} from "ts-mls/message.js"
import {protocolVersions} from "ts-mls/protocolVersion.js"
import {base64ToBytes} from "ts-mls/util/byteArray.js"
import {wireformats} from "ts-mls/wireformat.js"

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
	//
	// Encode the KeyPackage as an MLS message
	const keyPackageMessage = encode(mlsMessageEncoder, {
		keyPackage: publicPackage,
		wireformat: wireformats.mls_key_package,
		version: protocolVersions.mls10,
	})

	// TEST: Verify that we can decode the message we just encoded
	const keyPackageAsBase64 = bytesToBase64(keyPackageMessage)
	console.log("Created KeyPackage message as base64:", keyPackageAsBase64)
	const decodedMessage = decode(mlsMessageDecoder, base64ToBytes(keyPackageAsBase64))
	console.log("Decoded KeyPackage message:", decodedMessage)

	return {
		id: "", // This will be appened by the server
		type: "mls:KeyPackage",
		to: "as:Public",
		attributedTo: actorID,
		mediaType: "message/mls",
		encoding: "base64",
		generator: generator,
		content: keyPackageAsBase64,
	}
}
