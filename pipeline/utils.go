package pipeline

import "strings"

func getRepoName(repoURL string) string {
	repoSplit := strings.Split(repoURL, "/")
	return repoSplit[len(repoSplit)-1]
}
