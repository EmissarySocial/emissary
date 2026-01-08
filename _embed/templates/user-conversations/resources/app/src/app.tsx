import m from "mithril"

import {type APActor} from "./model/ap-actor"
import {Database, NewDatabase} from "./service/database"
import {Delivery} from "./service/delivery"
import {Directory} from "./service/directory"
import {loadActivityStream} from "./service/network"
import {NewMLS} from "./service/mls-loader"
import {Controller} from "./controller"
import {Main} from "./view/main"

async function startup() {
	// Collect arguments from the DOM
	const root = document.getElementById("mls")!
	const actorID = root.dataset["actor-id"] || ""

	// Verify that the root element exists
	if (root == undefined) {
		throw new Error(`Can't mount Mithril app. Please verify that <div id="mls"> exists.`)
	}

	// Load the actor object from the network
	const actor = (await loadActivityStream(actorID)) as APActor

	// Build dependencies
	const indexedDB = await NewDatabase()
	const database = new Database(indexedDB)
	const delivery = new Delivery(actor.id, actor.outbox)
	const directory = new Directory()
	const mls = await NewMLS(database, delivery, directory, actor)

	// Build the controller
	const controller = new Controller(actor, database, delivery, directory, mls)

	// Pass the controller to the Main component and mount the main application
	m.mount(root, {view: () => <Main controller={controller} />})
}

// 3..2..1.. Go!
startup()
