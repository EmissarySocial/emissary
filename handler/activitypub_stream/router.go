package activitypub_stream

import "github.com/benpate/hannibal/router"

// streamRouter defines the package-level router for stream/ActivityPub requests
var streamRouter = router.New[Context]()
