# Datastore Extensions for Go
This library extends the Go Google Cloud Datastore library with lower-level functionality that is only exposed on the gRPC interface.

When to use:

1. High performance use cases where every 10ms counts

When not to use:

1. Hands-off deployments that you expect to work in 10 years without any maintenance.
2. Low-staffed teams that do not have the bandwidth to investigate and fix OSS libaries (that they use) themselves.

Two examples:

1. Enforce relationship still exists between two entities retrieved separately - [example/parentchild](./example/parentchild)
1. Expose `version` externally to allow **optimistic transactions** spanning apps/clients - [example/externalzookie](./example/externalzookie)
