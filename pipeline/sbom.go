package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"dagger.io/dagger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	spdx "github.com/spdx/tools-golang/json"
	"github.com/spdx/tools-golang/spdx/common"
)

// GenerateSBOM generates a software bill of materials for the container image
func (p *Pipeline) GenerateSBOM(ctx context.Context, image string, objectStore string) (*dagger.FileID, error) {
	client := p.Client
	workdir := client.Host().Workdir()
	bom := client.Container().From("anchore/syft:latest")
	bom = bom.Exec(dagger.ContainerExecOpts{
		Args: []string{image, "--scope", "all-layers", "-o", "spdx-json", "--file", "sbom.json"},
	})

	fileID, err := bom.File("sbom.json").ID(ctx)
	if err != nil {
		return nil, err
	}

	dir, err := bom.Directory(".").ID(ctx)
	if err != nil {
		return nil, err
	}

	_, err = workdir.Write(ctx, dir, dagger.HostDirectoryWriteOpts{
		Path: "artifacts/",
	})
	if err != nil {
		return nil, err
	}
	if os.Getenv("SBOM_CREATOR") != "" {
		f, err := os.Open("artifacts/sbom.json")
		if err != nil {
			return nil, err
		}
		defer f.Close()

		doc, err := spdx.Load2_3(f)
		if err != nil {
			return nil, fmt.Errorf("unable to decode spdx-json: %w", err)
		}

		doc.CreationInfo.Creators = []common.Creator{
			{
				Creator:     os.Getenv("SBOM_CREATOR"),
				CreatorType: "Tool",
			},
		}

		// TODO: link to SBOM page
		doc.DocumentNamespace = "https://example.com/spdx.json"

		b, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile("artifacts/sbom.json", b, 0644)
		if err != nil {
			return nil, err
		}
	}

	if objectStore != "" && p.S3Client != nil {
		f, err := os.Open("artifacts/sbom.json")
		if err != nil {
			return nil, err
		}
		defer f.Close()
		_, err = p.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(objectStore),
			Key:    aws.String("sbom.json"),
			Body:   f,
		})
		if err != nil {
			return nil, err
		}
	}

	return &fileID, nil
}
