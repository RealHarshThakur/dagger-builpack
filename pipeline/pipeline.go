package pipeline

import (
	"context"
	"io"
	"os"

	"dagger.io/dagger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Pipeline contains fields/clients for building a pipeline
type Pipeline struct {
	Client *dagger.Client
	// Not including a logger here as this should remain a library and libraries should not log
	// Also, logging via the dagger client can be done if needed
	S3Client *s3.S3
}

// NewPipeline creates a new pipeline object
func NewPipeline(o io.Writer) (*Pipeline, error) {
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(o))
	if err != nil {
		return nil, err
	}

	var s3Client *s3.S3
	if os.Getenv("S3_ACCESS_KEY_ID") != "" && os.Getenv("S3_SECRET_ACCESS_KEY") != "" {
		creds := credentials.NewStaticCredentials(os.Getenv("S3_ACCESS_KEY_ID"), os.Getenv("S3_SECRET_ACCESS_KEY"), "")
		cfg := aws.NewConfig().WithRegion("us-east-1").
			WithEndpoint(os.Getenv("S3_API_ENDPOINT")).
			WithDisableSSL(false).
			WithLogLevel(3).
			WithS3ForcePathStyle(true).
			WithCredentials(creds)
		sess := session.Must(session.NewSession(
			&aws.Config{
				Region: aws.String("us-east-1"),
			},
		))
		s3Client = s3.New(sess, cfg)
	}

	return &Pipeline{
		Client:   client,
		S3Client: s3Client,
	}, nil
}
