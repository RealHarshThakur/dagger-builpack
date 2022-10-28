package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"dagger.io/dagger"
	"github.com/google/uuid"
)

// Pipeline contains fields/clients for building a pipeline
type Pipeline struct {
	Client *dagger.Client
}

// NewPipeline creates a new pipeline object
func NewPipeline() (*Pipeline, error) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(io.Discard))
	if err != nil {
		return nil, err
	}

	return &Pipeline{
		Client: client,
	}, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("must pass in a git repo to build")
		os.Exit(1)
	}
	repo := os.Args[1]
	p, err := NewPipeline()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx := context.Background()
	fmt.Printf("Building image %s", repo)
	image, err := p.Build(ctx, repo)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Built image %s\n", *image)

	fmt.Println("Generating SBOM for image", *image)
	sbom, err := p.GenerateSBOM(ctx, *image)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Generated SBOM for image", *image)

	fmt.Println("Scanning SBOM for vulnerabilities")
	res, err := p.ScanVulns(ctx, *sbom)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Scanned SBOM for vulnerabilities")
	fmt.Println(*res)
}

func getRepoName(repoURL string) string {
	repoSplit := strings.Split(repoURL, "/")
	return repoSplit[len(repoSplit)-1]
}

// Build builds a container image for the git repo using buildpacks
// If build is successful, returns the image name
func (p *Pipeline) Build(ctx context.Context, repoURL string) (*string, error) {
	client := p.Client
	imageTag := fmt.Sprintf("heroku/buildpacks:20")
	packBuilder := client.Container().From(imageTag)

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

// GenerateSBOM generates a software bill of materials for the container image
func (p *Pipeline) GenerateSBOM(ctx context.Context, image string) (*string, error) {
	client := p.Client
	bom := client.Container().From("anchore/syft:latest")
	bom = bom.Exec(dagger.ContainerExecOpts{
		Args: []string{image, "-o", "spdx-json"},
	})

	sbom, err := bom.Stdout().Contents(ctx)
	if err != nil {
		return nil, err
	}

	return &sbom, nil
}

// ScanVulns scans the SBOM for vulnerabilities
func (p *Pipeline) ScanVulns(ctx context.Context, sbom string) (*string, error) {
	client := p.Client
	scanner := client.Container().From("anchore/grype:latest")
	scanner = scanner.WithWorkdir("/tmp")

	scanner = scanner.Exec(dagger.ContainerExecOpts{
		Args: []string{"echo", sbom, ">", "/tmp/sbom.json"},
	})

	scanner = scanner.Exec(dagger.ContainerExecOpts{
		Args: []string{"sbom:/tmp/sbom.json > /tmp/vuln.json"},
	})

	result, err := scanner.File("/tmp/vuln.json").Contents(ctx)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
