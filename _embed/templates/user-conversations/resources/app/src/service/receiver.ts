import type {MlsPrivateMessage} from "ts-mls"
import {rangeCollection} from "./network"
import type {APMLSMessage} from "../model/ap-mlsmessage"
import * as ap from "../ap/properties"

// MessageHandler is a function that takes an MlsPrivateMessage and returns void.
// The Receiver service will call all registered MessageHandlers when a new message
// is received.
type MessageHandler = (message: string) => Promise<void>

// Receiver service receives messages from an ActivityPub actor and forwards them
// to the MLS client
export class Receiver {
	//

	#actorId: string // ID of the user receiving messages
	#messagesUrl: string // endpoint for the actor's mls:messages collection
	handler: MessageHandler // list of registered message handlers

	constructor(actorId: string, messagesUrl: string) {
		this.#actorId = actorId
		this.#messagesUrl = messagesUrl
		this.handler = async function (message: string) {
			console.log("Received message:", message)
		}
	}

	// registerHandler adds a new MessageHandler to the list of handlers that will be called
	registerHandler(handler: MessageHandler) {
		this.handler = handler
	}

	// start begins polling for new messages and processing them with the registered handlers
	// TODO: If the collection contains an SSE channel, then also start an SSE listener
	start() {
		console.log("starting receiver for actor:", this.#actorId)
		this.poll()
	}

	// poll retrieves new messages from the mls:messages collection and calls the
	// onMessage callback for each new message
	async poll() {
		const generator = rangeCollection<APMLSMessage>(this.#messagesUrl)
		for await (const message of generator) {
			console.log("Receiver: Received message:", message)
			const content = ap.Content(message)
			await this.handler(content)
		}
	}
}
