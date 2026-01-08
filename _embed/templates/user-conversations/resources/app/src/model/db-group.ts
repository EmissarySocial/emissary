import {type ClientState} from "ts-mls"

// DBGroup represents a group record in the indexedDB database
export type DBGroup = {
	groupID: string
	members: string[]
	name: string
	groupState: ClientState
	createDate: number
	updateDate: number
	readDate: number
}
