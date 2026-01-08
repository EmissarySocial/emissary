export type IDBMessage = {
	id: string
	groupId: string
	tyepe:"Note" | "Article"
	attributedTo: string
	to: string[]
	published: number
}