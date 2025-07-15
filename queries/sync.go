package queries

func SyncMongoDBIndexes(connectionString string, databaseName string) error {

	/*
		// Connect to the database
		ctx := context.Background()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))

		if err != nil {
			return derp.Wrap(err, "data.mongodb.New", "Error creating mongodb client")
		}

		session := client.Database(databaseName)

		// Sync the User collection indexes
		log.Debug().Msg("Sync Indexes: User Collection...")
		if err := sync.User(ctx, session); err != nil {
			return err
		}
	*/

	return nil
}
