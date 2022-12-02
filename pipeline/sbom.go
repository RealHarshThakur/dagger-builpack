package pipeline

import (
	"context"

	"dagger.io/dagger"
)

// GenerateSBOM generates a software bill of materials for the container image
func (p *Pipeline) GenerateSBOM(ctx context.Context, image string) (*dagger.FileID, error) {
	client := p.Client
	workdir := client.Host().Workdir()
	bom := client.Container().From("anchore/syft:latest")
	bom = bom.Exec(dagger.ContainerExecOpts{
		Args: []string{image, "-o", "spdx-json", "--file", "sbom.json"},
	})

	fileID, err := bom.File("sbom.json").ID(ctx)
	if err != nil {
		return nil, err
	}

	dir, err := bom.Directory(".").ID(ctx)
	if err != nil {
		return nil, err
	}

	_, err = workdir.Write(ctx, dir)
	if err != nil {
		return nil, err
	}
	return &fileID, nil
}
