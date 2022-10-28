package main

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("must pass in a git repo to build")
		os.Exit(1)
	}
	repo := os.Args[1]
	if err := build(repo); err != nil {
		fmt.Println(err)
	}
}

func build(repoURL string) error {
	fmt.Printf("Building %s\n", repoURL)

	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}
	defer client.Close()

	imageTag := fmt.Sprintf("heroku/buildpacks:20")
	packBuilder := client.Container().From(imageTag)

	packBuilder = packBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"git", "clone", repoURL, "/tmp/src"},
	})

	packBuilder = packBuilder.WithWorkdir("/tmp/src")

	build := packBuilder.Exec(dagger.ContainerExecOpts{
		Args: []string{"/cnb/lifecycle/creator", "-app=.", "ttl.sh/random-new-image:latest"},
	})

	_, err = build.Stdout().Contents(ctx)
	if err != nil {
		return err
	}

	return nil
}
