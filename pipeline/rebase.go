package pipeline

import (
	"context"

	"dagger.io/dagger"
)

// Rebase updates the runtime image with the latest security patches
func (p *Pipeline) Rebase(ctx context.Context, image string) error {
	client := p.Client
	packBuilder := client.Container().From("paketobuildpacks/builder:base")

	build := packBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"/cnb/lifecycle/rebaser", image},
	})

	_, err := build.Stdout().Contents(ctx)
	return err
}
