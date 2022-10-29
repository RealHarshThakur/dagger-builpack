package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"dagger.io/dagger"
	"github.com/anchore/grype/grype/presenter/models"
	"github.com/google/uuid"
)

// Pipeline contains fields/clients for building a pipeline
type Pipeline struct {
	Client *dagger.Client
}

// NewPipeline creates a new pipeline object
func NewPipeline() (*Pipeline, error) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
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
	fmt.Println("Generated SBOM for image, SBOM artifact stored in working directory: sbom.json", *image)

	fmt.Println("Scanning SBOM for vulnerabilities")
	err = p.GenerateVulnReport(ctx, *sbom)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Scanned SBOM for vulnerabilities, vulnerability report stored in working directory: vuln.json")
	vulns, err := ScanVuln()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	levels, fixes := parseVulnForSeverityLevels(vulns)
	for level, count := range levels {
		fmt.Printf("Found %d %s vulnerabilities\n", count, level)
	}
	fmt.Printf("%d vulnerabilities have fixes available\n", fixes)

}

func parseVulnForSeverityLevels(vulns []models.Vulnerability) (map[string]int, int) {
	levels := make(map[string]int, 0)
	fixes := 0
	for _, vuln := range vulns {
		if levels[vuln.Severity] == 0 {
			levels[vuln.Severity] = 1
		} else {
			levels[vuln.Severity]++
		}
		if len(vuln.Fix.Versions) > 0 {
			fixes++
		}
	}

	return levels, fixes
}

// ScanVuln scans the vuln report for vulnerabilities
func ScanVuln() ([]models.Vulnerability, error) {
	vulnJSON, err := os.ReadFile("./vuln.json")
	if err != nil {
		return nil, err
	}
	vulns := make([]models.Vulnerability, 0)
	doc := &models.Document{}
	err = json.Unmarshal(vulnJSON, &doc)
	if err != nil {
		return nil, err
	}

	for _, match := range doc.Matches {
		if match.Vulnerability.ID != "" {
			vulns = append(vulns, match.Vulnerability)
		}
	}

	return vulns, nil
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

// GenerateVulnReport scans the SBOM for vulnerabilities
func (p *Pipeline) GenerateVulnReport(ctx context.Context, file dagger.FileID) error {
	client := p.Client
	workdir := client.Host().Workdir()

	scanner := client.Container().From("anchore/grype:latest").
		WithMountedFile("/work/sbom.json", file)

	dir, err := scanner.Exec(dagger.ContainerExecOpts{
		Args: []string{"sbom:/work/sbom.json", "-o", "json", "--file", "vuln.json"},
	}).Directory(".").ID(ctx)
	if err != nil {
		return err
	}

	_, err = workdir.Write(ctx, dir)
	if err != nil {
		return err
	}

	return nil
}
