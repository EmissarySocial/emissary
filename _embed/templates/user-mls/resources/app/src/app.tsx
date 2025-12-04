import m from "mithril"
import { ViewContainer } from "./viewContainer"

class Application {

	constructor() {

	}
}

function run() {
	var root = document.getElementById("mls")

	if (root == undefined) {
		console.log("Cannot mount Mithril app. Please check that id=mls exists")
		return
	}

	var viewContainer = new ViewContainer()
	m.mount(root, viewContainer)
}

run()