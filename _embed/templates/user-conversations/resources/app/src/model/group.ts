import {type ClientState} from "ts-mls"

// Group represents a group record in memory
export type Group = {
	groupID: string
	members: string[]
	name: string
	clientState: ClientState
	createDate: number
	updateDate: number
	readDate: number
}
