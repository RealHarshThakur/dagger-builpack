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

	repo := client.Git(repoURL)
	src, err := repo.Branch("main").Tree().ID(ctx)
	if err != nil {
		return err
	}

	workdir := client.Host().Workdir()
	imageTag := fmt.Sprintf("heroku/buildpacks:20")
	heroku := client.Container().From(imageTag)
	heroku = heroku.WithMountedDirectory("/src", src).WithWorkdir("/src")

	build := heroku.Exec(dagger.ContainerExecOpts{
		Args: []string{"/cnb/lifecycle/creator", "-app=.", "ttl.sh/random-new-image:latest"},
	})

	output, err := build.Directory("/src").ID(ctx)
	if err != nil {
		return err
	}

	_, err = workdir.Write(ctx, output, dagger.HostDirectoryWriteOpts{Path: "./output"})
	if err != nil {
		return err
	}

	return nil
}
