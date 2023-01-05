package pipeline

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"dagger.io/dagger"
	"github.com/google/uuid"
)

type DockerConfig struct {
	Auths map[string]AuthConfig `json:"auths"`
}

type AuthConfig struct {
	Auth  string `json:"auth"`
	Email string `json:"email"`
}

type RegistryInfo struct {
	RegistryServer   string
	RegistryUsername string
	RegistryPassword string
	RegistryEmail    string
	RepoName         string
	ImageName        string
	ImageTag         string
}

func NewDockerConfig(username, password, email, server string) *DockerConfig {
	if username == "" || password == "" || email == "" || server == "" {
		return nil
	}
	authStr := fmt.Sprintf("%s:%s", username, password)
	authBytes := []byte(authStr)
	encodedAuth := base64.StdEncoding.EncodeToString(authBytes)

	return &DockerConfig{
		Auths: map[string]AuthConfig{
			fmt.Sprintf("https://%s", server): {
				Auth:  encodedAuth,
				Email: email,
			},
		},
	}
}

// Build builds a container image for the git repo using buildpacks
// If build is successful, returns the image name
func (p *Pipeline) Build(ctx context.Context, buildTool, repoURL, builderImage string, regInfo *RegistryInfo) (*string, error) {

	switch buildTool {
	case "ko":
		return p.KoBuild(ctx, repoURL, regInfo)
	}
	return p.PackBuild(ctx, builderImage, repoURL, regInfo)
}

func generateDC(regInfo *RegistryInfo) ([]byte, error) {
	var dcBytes []byte
	if regInfo != nil {
		var err error
		dc := NewDockerConfig(regInfo.RegistryUsername, regInfo.RegistryPassword, regInfo.RegistryEmail, regInfo.RegistryServer)
		if dc != nil {
			dcBytes, err = json.Marshal(dc)
			if err != nil {
				return nil, err
			}
		}
	}
	return dcBytes, nil
}

func (p *Pipeline) KoBuild(ctx context.Context, repoURL string, regInfo *RegistryInfo) (*string, error) {
	client := p.Client
	repoName := getRepoName(repoURL)

	dcBytes, err := generateDC(regInfo)
	if err != nil {
		return nil, err
	}

	if len(dcBytes) == 0 {
		regInfo = &RegistryInfo{}
		regInfo.RegistryServer = "ttl.sh"
		regInfo.RepoName = repoName
		regInfo.ImageName = fmt.Sprintf("%s-%s", repoName, uuid.New().String()[:5])
		regInfo.ImageTag = "60m"
	}

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

	ddir := client.Directory().WithNewFile("/tmp/config.json", dagger.DirectoryWithNewFileOpts{
		Contents: string(dcBytes),
	})

	koBuilder = koBuilder.WithWorkdir("/tmp/src")

	ddirID, err := ddir.ID(ctx)
	if err != nil {
		return nil, err
	}

	koBuilder = koBuilder.WithMountedDirectory("/mnt", ddirID)

	imageName := fmt.Sprintf("%s/%s/%s", regInfo.RegistryServer, regInfo.RepoName, regInfo.ImageName)
	build := koBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"sh", "-c", "mkdir -p /root/.docker/"},
	}).Exec(dagger.ContainerExecOpts{
		Args: []string{"sh", "-c", "cp /mnt/tmp/config.json /root/.docker/config.json"},
	}).
		Exec(dagger.ContainerExecOpts{
			Args: []string{"sh", "-c", fmt.Sprintf("KO_DOCKER_REPO=%s ko build . --bare -t %s", imageName, regInfo.ImageTag)},
		})

	imageWithTag := fmt.Sprintf("%s:%s", imageName, regInfo.ImageTag)
	_, err = build.Stdout().Contents(ctx)
	return &imageWithTag, err
}

func (p *Pipeline) PackBuild(ctx context.Context, builderImage, repoURL string, regInfo *RegistryInfo) (*string, error) {
	client := p.Client
	packBuilder := client.Container().From(builderImage)
	repoName := getRepoName(repoURL)

	dcBytes, err := generateDC(regInfo)
	if err != nil {
		return nil, err
	}

	if len(dcBytes) == 0 {
		regInfo = &RegistryInfo{}
		regInfo.RegistryServer = "ttl.sh"
		regInfo.RepoName = repoName
		regInfo.ImageName = fmt.Sprintf("%s-%s", repoName, uuid.New().String()[:5])
		regInfo.ImageTag = "60m"
	}

	ddir := client.Directory().WithNewFile("/tmp/config.json", dagger.DirectoryWithNewFileOpts{
		Contents: string(dcBytes),
	})

	ddirID, err := ddir.ID(ctx)
	if err != nil {
		return nil, err
	}

	packBuilder = packBuilder.WithMountedDirectory("/mnt", ddirID)

	packBuilder = packBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"git", "clone", repoURL, "/tmp/src"},
	})

	packBuilder = packBuilder.WithWorkdir("/tmp/src")

	imageName := fmt.Sprintf("%s/%s/%s:%s", regInfo.RegistryServer, regInfo.RepoName, regInfo.ImageName, regInfo.ImageTag)
	packBuilder = packBuilder.Exec(
		dagger.ContainerExecOpts{
			Args: []string{"mkdir", "-p", "/home/cnb/.docker/"},
		}).Exec(dagger.ContainerExecOpts{
		Args: []string{"sh", "-c", "cp /mnt/tmp/config.json /home/cnb/.docker/config.json"},
	})
	build := packBuilder.Exec(
		dagger.ContainerExecOpts{
			Args: []string{"bash", "-c", fmt.Sprintf("CNB_PLATFORM_API=0.8 /cnb/lifecycle/creator -app=. %s", imageName)},
		})

	_, err = build.Stdout().Contents(ctx)
	return &imageName, err
}
