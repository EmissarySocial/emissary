package consumer

/*
func UpdateContext(serverFactory ServerFactory, args mapof.Any) queue.Result {

	const location = "consumer.UpdateContext"

	oldContext := args.GetString("oldContext")
	newContext := args.GetString("newContext")

	database := serverFactory.CommonDatabase()
	collection := database.Collection("Document")

	if err := queries.UpdateContext(collection, oldContext, newContext); err != nil {
		err = derp.Wrap(err, location, "Unable to update context in Document collection")
		return queue.Error(err)
	}

	return queue.Success()
}
*/
