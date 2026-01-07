export function rootNode(): HTMLElement {
	const root = document.getElementById("mls")

	if (root == undefined) {
		throw new Error(`Can't find root node: <div id="mls"></div>. Please verify that it exists.`)
	}

	return root
}

export function myActorID(): string {
	return rootNode().dataset["actor-id"] || ""
}

export function myOutboxID(): string {
	return rootNode().dataset["outbox-id"] || ""
}
