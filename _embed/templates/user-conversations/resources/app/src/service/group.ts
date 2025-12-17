import type { IDBPDatabase } from "idb";
import type { Group } from "../model/group";
import type { ActivityPubService } from "./activityPub";

export class GroupService {

	#activityPubService: ActivityPubService
	#database: IDBPDatabase
	#actorID: string

	constructor(actorID: string, activityPubService: ActivityPubService, database: IDBPDatabase) {
		this.#activityPubService = activityPubService
		this.#database = database
		this.#actorID = actorID
	}

	/*
	public async create(): Promise<[Group, string]> {

		// Create the group in the local IndexedDB
		var group: Group = {
			actors: [this.#actorID],
		}

		// Publish the group to ActivityPub
		var [groupInfo, err] = await this.#activityPubService.createObject(groupInfo)

		if (err != "") {
			return [group, err]
		}

		return group, ""
	}
	*/
}

