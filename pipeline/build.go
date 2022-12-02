package pipeline

import (
	"context"
	"fmt"

	"dagger.io/dagger"
	"github.com/google/uuid"
)

// Build builds a container image for the git repo using buildpacks
// If build is successful, returns the image name
func (p *Pipeline) Build(ctx context.Context, builderImage, repoURL string) (*string, error) {
	client := p.Client
	packBuilder := client.Container().From(builderImage)

	packBuilder = packBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"git", "clone", repoURL, "/tmp/src"},
	})

	packBuilder = packBuilder.WithWorkdir("/tmp/src")

	repoName := getRepoName(repoURL)
	imageName := fmt.Sprintf("ttl.sh/%s-%s:30m", repoName, uuid.New().String()[:5])
	build := packBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"/cnb/lifecycle/creator", "-app=.", imageName},
	})

	_, err := build.Stdout().Contents(ctx)
	return &imageName, err
}
