import m from "mithril"

import { ServiceFactory } from "./service/factory"
import { ViewContainer } from "./viewContainer"

class Application {

	constructor(root:HTMLElement) {
		this.start(root)
	}

	private async start(root:HTMLElement) {

		var factory = new ServiceFactory()
		await factory.start()

		var viewContainer = new ViewContainer(factory)
		m.mount(root, viewContainer)
	}
}

// Start the Application
var app: Application
var root = document.getElementById("mls")

if (root != undefined) {
	const app = new Application(root)
} else {
	console.log("Can't mount Mithril app. Please verify that <div id=mls> exists.")
}