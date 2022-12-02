package pipeline

import (
	"context"
	"io"

	"dagger.io/dagger"
)

// Pipeline contains fields/clients for building a pipeline
type Pipeline struct {
	Client *dagger.Client
	// Not including a logger here as this should remain a library and libraries should not log
	// Also, logging via the dagger client can be done if needed
}

// NewPipeline creates a new pipeline object
func NewPipeline(o io.Writer) (*Pipeline, error) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(o))
	if err != nil {
		return nil, err
	}

	return &Pipeline{
		Client: client,
	}, nil
}
