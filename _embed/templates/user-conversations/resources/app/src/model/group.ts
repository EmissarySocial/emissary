import {type ClientState} from "ts-mls"

// Group represents a group record in memory
export type Group = {
	id: string
	name: string
	members: string[]
	clientState: ClientState
	createDate: number
	updateDate: number
	readDate: number
}
