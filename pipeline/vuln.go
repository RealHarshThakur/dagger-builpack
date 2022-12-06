package pipeline

import (
	"context"
	"encoding/json"
	"os"

	"dagger.io/dagger"
	"github.com/anchore/grype/grype/presenter/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// ParseVulnForSeverity parses the vuln report for a specific severity level and returns count of each level
func ParseVulnForSeverityLevels(vulns []models.Vulnerability) (map[string]int, int) {
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
	vulnJSON, err := os.ReadFile("./artifacts/vuln.json")
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

// GenerateVulnReport scans the SBOM for vulnerabilities
func (p *Pipeline) GenerateVulnReport(ctx context.Context, file dagger.FileID, objectStore string) error {
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

	_, err = workdir.Write(ctx, dir, dagger.HostDirectoryWriteOpts{
		Path: "artifacts/",
	})
	if err != nil {
		return err
	}

	if objectStore != "" && p.S3Client != nil {
		f, err := os.Open("artifacts/vuln.json")
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = p.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(objectStore),
			Key:    aws.String("vuln.json"),
			Body:   f,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
