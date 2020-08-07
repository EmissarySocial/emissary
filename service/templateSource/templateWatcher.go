package templateSource

/*
// TemplateWatcher initiates a mongodb change stream to on every updates to Template data objects
// TODO: this will be moved into the mongodb code
func TemplateWatcher(uri string, database string) chan model.Template {

	result := make(chan model.Template)

	ctx := context.Background()

	client, err := mongo.NewClient(options.Client().ApplyURI(uri))

	if err != nil {
		derp.Report(err)
		return result
	}

	if err := client.Connect(ctx); err != nil {
		derp.Report(err)
		return result
	}

	collection := client.Database(database).Collection("Template")

	cs, err := collection.Watch(ctx, mongo.Pipeline{})

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.service.Watcher", "Unable to open Mongodb Change Template"))
		return result
	}

	go func() {

		for cs.Next(ctx) {

			var event struct {
				Template model.Template `bson:"fullDocument"`
			}

			if err := cs.Decode(&event); err != nil {
				derp.Report(err)
				continue
			}

			spew.Dump("Watcher. Writing stream to channel.", event.Template)
			result <- event.Template
		}
	}()

	return result
}

*/
