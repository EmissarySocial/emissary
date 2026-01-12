import {type KeyPackage} from "ts-mls"
import {decodeMlsMessage} from "ts-mls"
import {loadActor} from "./actor"
import {rangeCollection} from "./collection"
import {type MLSKeyPackageBundle} from "../z-MLS/MLSManager"

// KeyPackage is the ActivityPub representation of a KeyPackage
// https://swicg.github.io/activitypub-e2ee/mls#KeyPackage
export type APKeyPackage = {
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

// NewAPKeyPackage creates a fully initialized KeyPackage object
export function NewAPKeyPackage(actorID: string, publicPackage: KeyPackage): APKeyPackage {
	return {
		id: "", // This will be appened by the server
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

export async function getKeyPackages(recipients: string[]): Promise<MLSKeyPackageBundle[]> {
	var result = [] as MLSKeyPackageBundle[]

	for (const recipient of recipients) {
		const keyPackages = await getKeyPackagesForActor(recipient)
		for (const keyPackage of keyPackages) {
			result.push(keyPackage)
		}
	}

	return result
}

export async function getKeyPackagesForActor(actorID: string): Promise<MLSKeyPackageBundle[]> {
	console.log("Loading KeyPackages for actor", actorID)
	const actor = await loadActor(actorID)
	console.log(actor)
	const rangeKeyPackages = rangeCollection<APKeyPackage>(actor.keyPackages)

	var result: MLSKeyPackageBundle[] = []

	/*
	for await (const collectionItem of rangeKeyPackages) {
		console.log("KeyPackage item", collectionItem)
		const contentBytes = base64ToUint8Array(collectionItem.content)
		const mlsMessage = decodeMlsMessage(contentBytes, 0)![0]

		keyPackage = mlsMessage.

		const bundle: MLSKeyPackageBundle = {
			userId: actorID,
			publicPackage: keyPackage[0],
		}

		result.push(bundle)
	}
	*/

	return result
}

function base64ToUint8Array(base64: string): Uint8Array {
	const binary_string = window.atob(base64)
	const len = binary_string.length
	const bytes = new Uint8Array(len)
	for (let i = 0; i < len; i++) {
		bytes[i] = binary_string.charCodeAt(i)
	}
	return bytes
}
