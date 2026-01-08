import {type DBGroup} from "../model/db-group"
import {type KeyPackage, type Welcome} from "ts-mls"
import {type MLSMessage} from "ts-mls/message.js"

// IDatabase wraps all of the methods that the MLS service
// uses to store group state.
export interface IDatabase {
	saveGroup(group: DBGroup): Promise<void>
	loadGroup(groupID: string): Promise<DBGroup>
}

// IDelivery wraps all of the methods that the MLS service
// uses to send messages.
export interface IDelivery {
	sendWelcome(recipients: string[], welcome: Welcome): Promise<void>
	sendCommit(recipients: string[], commit: MLSMessage): Promise<void>
}

// IDirectory wraps all of the methods that the MLS service
// uses to look up users' KeyPackages.
export interface IDirectory {
	getKeyPackages(actorIDs: string[]): Promise<KeyPackage[]>
}
