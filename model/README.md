# model

This package defines Emissary's domain data structures — the objects that are stored in the database and passed through the application: `Stream`, `User`, `Follower`, `Following`, `Rule`, `Conversation`, `KeyPackage`, and many more. Each model carries its accessors and the constants that describe its states and fields, but no business logic — that lives in the [service](../service/) layer.

The [step](step/) subdirectory holds the data definitions for each builder pipeline step (the building functions themselves live in [build](../build/)). See the [project README](../README.md) for the big picture.
