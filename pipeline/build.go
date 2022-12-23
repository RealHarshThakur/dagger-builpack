package pipeline

import (
	"context"
	"fmt"

	"dagger.io/dagger"
	"github.com/google/uuid"
)

// Build builds a container image for the git repo using buildpacks
// If build is successful, returns the image name
func (p *Pipeline) Build(ctx context.Context, buildTool, repoURL, builderImage string) (*string, error) {
	switch buildTool {
	case "ko":
		return p.KoBuild(ctx, repoURL)
	}
	return p.PackBuild(ctx, builderImage, repoURL)
}

func (p *Pipeline) KoBuild(ctx context.Context, repoURL string) (*string, error) {
	client := p.Client
	repoName := getRepoName(repoURL)

	koBuilder := client.Container().From("golang:1.20rc1-alpine3.17")
	koBuilder = koBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"sh", "-c", "echo https://dl-cdn.alpinelinux.org/alpine/edge/testing/ >> /etc/apk/repositories"},
	}).Exec(dagger.ContainerExecOpts{
		Args: []string{"apk", "update"},
	}).Exec(dagger.ContainerExecOpts{
		Args: []string{"apk", "add", "git"},
	}).Exec(dagger.ContainerExecOpts{
		Args: []string{"apk", "add", "ko"},
	}).Exec(dagger.ContainerExecOpts{
		Args: []string{"git", "clone", repoURL, "/tmp/src"},
	})

	koBuilder = koBuilder.WithWorkdir("/tmp/src")

	imageName := fmt.Sprintf("ttl.sh/%s-%s", repoName, uuid.New().String()[:5])
	build := koBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"sh", "-c", fmt.Sprintf("KO_DOCKER_REPO=%s ko build . --bare -t 30m", imageName)},
	})

	imageWithTag := fmt.Sprintf("%s:30m", imageName)
	_, err := build.Stdout().Contents(ctx)
	return &imageWithTag, err
}

func (p *Pipeline) PackBuild(ctx context.Context, builderImage, repoURL string) (*string, error) {
	client := p.Client
	packBuilder := client.Container().From(builderImage)

	packBuilder = packBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"git", "clone", repoURL, "/tmp/src"},
	})

	packBuilder = packBuilder.WithWorkdir("/tmp/src")

	repoName := getRepoName(repoURL)
	imageName := fmt.Sprintf("ttl.sh/%s-%s:30m", repoName, uuid.New().String()[:5])
	build := packBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"bash", "-c", fmt.Sprintf("CNB_PLATFORM_API=0.8 /cnb/lifecycle/creator -app=. %s", imageName)},
	})

	_, err := build.Stdout().Contents(ctx)
	return &imageName, err
}
