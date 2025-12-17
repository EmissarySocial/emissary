import { ActivityPubService } from "./activityPub"
import { KeyPackageService } from "./keyPackage"
import { type APActor } from "../model/actor"

export class ServiceFactory {

	// All class #properties are PRIVATE
	#actor: APActor | {} = {}
	#activityPub: ActivityPubService
	#keyPackage: KeyPackageService

	constructor() {
		this.#activityPub = new ActivityPubService()
		this.#keyPackage = new KeyPackageService(this.#activityPub)
	}

	async start() {
		const actor = await this.loadActor()
		this.#actor = actor

		await this.#activityPub.start(actor.id)
		await this.#keyPackage.start(actor.id)
	}

	async loadActor(): Promise<APActor> {

		const response = await fetch("http://localhost/@me", {
			headers: [["Accept", "application/json"]],
			credentials: "same-origin",
		})

		const result = await response.json()

		if (typeof result == "object") {
			return result
		}

		return {
			id:"",
			name:"",
			inbox:"",
			keyPackages: {
				type:"Collection",
				id:"",
				items:[]
			}
		}
	}
}