import {type ClientState} from "ts-mls"
import {type DBGroup} from "./db-group"

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
