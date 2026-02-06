import {type ClientState} from "ts-mls"

// DBGroup represents a group record in the indexedDB database
export type DBGroup = {
	id: string
	name: string
	members: string[]
	clientState: ClientState
	createDate: number
	updateDate: number
	readDate: number
}
