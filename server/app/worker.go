package app

import (
	"context"

	"go.uber.org/dig"
)

// Worker represents an interface for a long-running background process.
type Worker interface {
	Run(ctx context.Context)
}

// RegisterWorker registers a Worker constructor to the dig container under the "workers" group.
func RegisterWorker(constructor any) {
	Provide(constructor, dig.Group("workers"))
}
