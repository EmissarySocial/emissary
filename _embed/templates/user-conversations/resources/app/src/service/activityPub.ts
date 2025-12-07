// ActivityPubService manages all interactions with the ActivityPub server
export class ActivityPubService {

	// All class #properties are PRIVATE
	#actorID: string = "" 

	async start(actorID: string) {
		this.#actorID = actorID
	}

	async createObject<T>(object: T): Promise<[T, string]> {

		const [result, err] = await this.sendActivity({
			"@context": "",
			"id": "",
			"type": "Create",
			"actor": this.#actorID,
			"object": object,
		})

		return [<T>result, err]
	}

	async deleteObject(objectId: string): Promise<string> {

		const [_result, err] = await this.sendActivity({
			"@context": "",
			"id": "",
			"type": "Delete",
			"actor": this.#actorID,
			"object": objectId,
		})

		return err
	}

	async sendActivity<T>(activity: Object): Promise<[T, string]> {

		try {

			// Send the Activity to the server
			const result = await fetch("http://localhost/@me/outbox", {
				method: "POST",
				body: JSON.stringify(activity),
				credentials: "include",
			})

			const resultObject = await result.json()

			return [resultObject, ""]

		} catch (err) {
			return [<T>{}, String(err)]
		}
	}
}