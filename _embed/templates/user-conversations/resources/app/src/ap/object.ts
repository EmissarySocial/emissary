//// Activities

type Activity = {
	type: string
	actor: string
}

type Accept = Activity & {
	type: "Accept"
	object: Activity
}

type Create = Activity & {
	type: "Create"
	object: Object
}

type Delete = Activity & {
	type: "Delete"
	object: string
}

type Like = Activity & {
	type: "Like"
	object: string
}

type Undo = Activity & {
	type: "Undo"
	object: Activity
}

type Update = Activity & {
	type: "Update"
	object: Object
}

//// Objects

type Object = {
	type: string
	id: string
}

type Note = Object & {
	type: "Note"
	content: string
}

type Article = Object & {
	type: "Article"
	content: string
}

type Image = Object & {
	type: "Image"
	url: string
}

//// Complete Actions

type CreateNote = Create & {
	object: Note
}

type CreateArticle = Create & {
	object: Article
}

type UpdateNote = Update & {
	object: Note
}

type UpdateArticle = Update & {
	object: Article
}

type DeleteNote = Delete & {
	object: string
}

type DeleteArticle = Delete & {
	object: string
}
