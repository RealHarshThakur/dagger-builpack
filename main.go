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
	gitURL, builderImage string
)

func init() {
	flag.StringVar(&gitURL, "git-url", "", "git url to build")
	flag.StringVar(&gitURL, "g", "", "git url to build")

	flag.StringVar(&builderImage, "builder-image", "paketobuildpacks/builder:base", "builder image to use")
	flag.StringVar(&builderImage, "b", "paketobuildpacks/builder:base", "builder image to use")

}
func main() {
	flag.Parse()
	log := SetupLogging()

	if gitURL == "" {
		log.Fatal("git-url is required")
	}

	repo := gitURL
	p, err := pl.NewPipeline(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	log.Infof("Building image %s", repo)
	image, err := p.Build(ctx, builderImage, repo)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Built image %s\n", *image)

	log.Infof("Generating SBOM for image", *image)

	sbom, err := p.GenerateSBOM(ctx, *image)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Generated SBOM for image, SBOM artifact stored in working directory: sbom.json", *image)

	log.Infof("Scanning SBOM for vulnerabilities %s", &image)
	err = p.GenerateVulnReport(ctx, *sbom)
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
