import {type KeyPackage} from "ts-mls"
import {decodeMlsMessage} from "ts-mls"
import {type APActor} from "../model/ap-actor"
import {type APKeyPackage} from "../model/ap-keypackage"
import {loadActivityStream} from "./network"
import {rangeCollection} from "./network"
import {base64ToUint8Array} from "./utils"

export class Directory {
	//

	// getKeyPackage loads the KeyPackages published by a single actor
	async getKeyPackages(actorIDs: string[]): Promise<KeyPackage[]> {
		var result: KeyPackage[] = []

		for (const actorID of actorIDs) {
			console.log("getKeyPackages: Loading KeyPackages for: " + actorID)
			const actor = (await loadActivityStream(actorID)) as APActor
			console.log(actor)
			const rangeKeyPackages = rangeCollection<APKeyPackage>(actor.keyPackages)

			for await (const item of rangeKeyPackages) {
				console.log("KeyPackage item", item)
				const contentBytes = base64ToUint8Array(item.content)
				const mlsMessage = decodeMlsMessage(contentBytes, 0)![0]

				if (mlsMessage.wireformat != "mls_key_package") {
					throw new Error("Invalid KeyPackage message")
				}

				result.push(mlsMessage.keyPackage)
			}
		}

		return result
	}

	// createKeyPackage publishes a new KeyPackage to the User's outbox.
	async createKeyPackage(keyPackage: APKeyPackage): Promise<string> {
		return ""
	}
}
