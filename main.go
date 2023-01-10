package main

import (
	"context"
	"flag"
	"os"
	"path"
	"runtime"
	"strconv"

	pl "github.com/RealHarshThakur/dagger-buildpack/pipeline"
	"github.com/sirupsen/logrus"
)

var (
	gitURL, gitBranch, gitTag, onlyBuild, builderImage, rebaseImage, objectStoreName, buildTool, registryServer, registryUsername, registryPassword, registryEmail, registryRepoName, registryImageName, registryImageTag string
)

func init() {
	flag.StringVar(&rebaseImage, "rebase-image", "", "The image to rebase")
	flag.StringVar(&rebaseImage, "r", "", "The image to rebase")

	flag.StringVar(&gitURL, "git-url", "", "git url to build")
	flag.StringVar(&gitURL, "g", "", "git url to build")

	flag.StringVar(&gitBranch, "git-branch", "", "git branch to build")

	flag.StringVar(&gitTag, "git-tag", "", "git tag to build")

	flag.StringVar(&builderImage, "builder-image", "paketobuildpacks/builder:base", "builder image to use")
	flag.StringVar(&builderImage, "b", "paketobuildpacks/builder:base", "builder image to use")

	flag.StringVar(&buildTool, "build-tool", "pack", "build tool to use")
	flag.StringVar(&buildTool, "t", "pack", "build tool to use")

	flag.StringVar(&objectStoreName, "object-store-name", "", "object store name")
	flag.StringVar(&objectStoreName, "o", "", "object store name")

	flag.StringVar(&registryServer, "registry-server", "", "registry server")
	flag.StringVar(&registryServer, "s", "", "registry server")

	flag.StringVar(&registryUsername, "registry-username", "", "registry username")
	flag.StringVar(&registryUsername, "u", "", "registry username")

	flag.StringVar(&registryPassword, "registry-password", "", "registry password")
	flag.StringVar(&registryPassword, "p", "", "registry password")

	flag.StringVar(&registryEmail, "registry-email", "", "registry email")
	flag.StringVar(&registryEmail, "e", "", "registry email")

	flag.StringVar(&registryRepoName, "registry-repo-name", "", "registry repo name")
	flag.StringVar(&registryRepoName, "n", "", "registry repo name")

	flag.StringVar(&registryImageName, "registry-image-name", "", "registry image name")
	flag.StringVar(&registryImageName, "i", "", "registry image name")

	flag.StringVar(&registryImageTag, "registry-image-tag", "latest", "registry image tag")
	flag.StringVar(&registryImageTag, "a", "latest", "registry image tag")

	flag.StringVar(&onlyBuild, "only-build", "false", "only build the image, do not generate SBOM/vuln report")
	flag.StringVar(&onlyBuild, "", "false", "only build the image")

}
func main() {
	flag.Parse()
	log := SetupLogging()

	dirPath := "artifacts"

	// Check if the directory already exists.
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Create the directory if it doesn't exist.
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	ctx := context.Background()

	p, err := pl.NewPipeline(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	if rebaseImage != "" {
		err = p.Rebase(ctx, rebaseImage)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Successfully rebased image %s", rebaseImage)
		os.Exit(0)
	}

	if gitURL == "" {
		log.Fatal("git-url is required")
	}

	if os.Getenv("GIT_TOKEN") == "" {
		log.Fatal("GIT_TOKEN is required")
	}

	repo := gitURL

	log.Infof("Building image %s", repo)
	var regInfo *pl.RegistryInfo
	if registryServer != "" && registryUsername != "" && registryPassword != "" && registryEmail != "" && registryRepoName != "" && registryImageName != "" {
		regInfo = &pl.RegistryInfo{
			RegistryServer:   registryServer,
			RegistryUsername: registryUsername,
			RegistryPassword: registryPassword,
			RegistryEmail:    registryEmail,
			RepoName:         registryRepoName,
			ImageName:        registryImageName,
			ImageTag:         registryImageTag,
		}
	}

	err = p.GitClone(repo, gitBranch, gitTag)
	if err != nil {
		log.Fatal(err)
	}

	image, err := p.Build(ctx, buildTool, repo, builderImage, regInfo)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Built image %s\n", *image)

	if onlyBuild == "true" {
		os.Exit(0)
	}

	log.Infof("Generating SBOM for image %s", *image)

	sbom, err := p.GenerateSBOM(ctx, *image, objectStoreName)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Generated SBOM for image %s", *image)

	log.Infof("Scanning SBOM for vulnerabilities %s", *image)
	err = p.GenerateVulnReport(ctx, *sbom, objectStoreName)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Scanned SBOM for vulnerabilities, vulnerability report stored in working directory: vuln.json")
	vulns, err := pl.ScanVuln()
	if err != nil {
		log.Fatal(err)
	}

	levels, fixes := pl.ParseVulnForSeverityLevels(vulns)
	for level, count := range levels {
		log.Infof("Found %d %s vulnerabilities\n", count, level)
	}
	log.Infof("%d vulnerabilities have fixes available\n", fixes)

}

// SetupLogging sets up the logging for the router daemon
func SetupLogging() *logrus.Logger {
	// Logging create logging object
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.DebugLevel)
	log.SetReportCaller(true)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := path.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
			return "", fileName
		},
	})

	return log
}
