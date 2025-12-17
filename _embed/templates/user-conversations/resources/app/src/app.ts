import m from "mithril"

import { ServiceFactory } from "./service/factory"
import { Main } from "./component/main"

class Application {

	constructor(root:HTMLElement) {
		this.start(root)
	}

	private async start(root:HTMLElement) {

		var factory = new ServiceFactory()
		await factory.start()

		var viewContainer = new Main(factory)
		m.mount(root, Main)
	}
}

// Start the Application
var root = document.getElementById("mls")

if (root != undefined) {
	const app = new Application(root)
} else {
	console.log("Can't mount Mithril app. Please verify that <div id=mls> exists.")
}