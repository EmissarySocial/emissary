import * as property from "./properties"

// Document is a wrapper around a JSON object that provides methods for accessing common ActivityPub properties
export class Document {
	#value: {[key: string]: any}

	constructor(value?: {[key: string]: any}) {
		this.#value = {}
		if (value != undefined) {
			this.#value = value
		}
	}

	//// Conversion methods

	// fromURL retrieves a JSON document from the specified URL and parses it into the Document struct
	async fromURL(url: string): Promise<Document> {
		const response = await fetch(url)
		this.fromJSON(await response.text())
		return this
	}

	// fromJSON parses a JSON string into the Document struct
	fromJSON(json: string): Document {
		this.#value = JSON.parse(json)
		return this
	}

	toObject(): {[key: string]: any} {
		return this.#value
	}

	//// Property accessors

	id(): string {
		return property.Id(this.#value)
	}

	actor(): string {
		return property.Actor(this.#value)
	}

	outbox(): string {
		return property.Outbox(this.#value)
	}

	type(): string {
		return property.Type(this.#value)
	}

	name(): string {
		return property.Name(this.#value)
	}

	summary(): string {
		return property.Summary(this.#value)
	}

	content(): string {
		return property.Content(this.#value)
	}

	eventStream(): string {
		return property.EventStream(this.#value)
	}

	mlsMessage(): string {
		return property.MlsMessage(this.#value)
	}

	mlsKeyPackages(): string {
		return property.MlsKeyPackages(this.#value)
	}
}
