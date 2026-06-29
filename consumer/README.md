# consumer

This package holds Emissary's background-task consumer. When work is published to the [Turbine](https://github.com/benpate/turbine) queue, the `Consumer` here picks it up and runs it — dispatching each task by name to the right handler (sending ActivityPub messages, crawling reply trees, moving users, connecting push services, and so on). It is the worker side of Emissary's queue: the [service](../service/) layer publishes tasks, and this package executes them off the request path.

See the [project README](../README.md) for the big picture.
