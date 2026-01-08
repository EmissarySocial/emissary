import m from "mithril"

import {NewDatabase, Database} from "./service/database"

// old imports
import {Main} from "./view/main"
import {MLSManager} from "./z-MLS/MLSManager"
import {type APActor} from "./z-activitypub/actor"
import {loadActor} from "./z-activitypub/actor"
import {getKeyPackages} from "./z-activitypub/keyPackage"
import {makeMLSService} from "./service/mls-loader"
import {type IDelivery, type IDirectory, type IDatabase} from "./service/interfaces"
import {MLS} from "./service/mls"
import {Controller} from "./controller"
import {Delivery} from "./service/delivery"
import {Directory} from "./service/directory"

async function NewController() {
	// Collect arguments from the DOM
	const root = document.getElementById("mls")!
	const actorID = root.dataset["actor-id"] || ""

	// Verify that the root element exists
	if (root == undefined) {
		throw new Error(`Can't mount Mithril app. Please verify that <div id="mls"> exists.`)
	}

	// Load the actor object from the network
	const actorResponse = await fetch(actorID)
	const actor = (await actorResponse.json()) as APActor

	// Build dependencies
	const indexedDB = await NewDatabase()
	const database = new Database(indexedDB)
	const delivery = new Delivery(actor.id, actor.outbox)
	const directory = new Directory()
	const mls = await makeMLSService(database, delivery, directory, actor.id)

	// Build the controller
	const controller = new Controller(actor, database, delivery, directory, mls)

	// Create and mount the main application
	m.mount(root, controller)
}

NewController()
