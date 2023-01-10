package pipeline

import (
	"fmt"
	"os"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// GitClone clones a git repo in current working directory(./src) and checks out the specified branch/tag
// Tag is higher priority than branch
// Defaults to main/master- main if both are found.
func (p *Pipeline) GitClone(repoURL, branchName, tag string) error {
	err := os.RemoveAll("./src")
	if err != nil {
		return err
	}

	repo, err := git.PlainClone("./src", false, &git.CloneOptions{
		URL: repoURL,
		Auth: &http.BasicAuth{
			Username: "abc123", // yes, this can be anything except an empty string
			Password: os.Getenv("GIT_TOKEN"),
		},
	})
	if err != nil {
		return err
	}

	var r *plumbing.Reference
	var branchFound, mainFound, masterFound bool
	branches, err := repo.Branches()
	if err != nil {
		return err
	}

	branches.ForEach(func(t *plumbing.Reference) error {
		if t.Name().IsBranch() {
			name := t.Name().Short()
			if branchName != "" && name == branchName {
				r = t
				branchFound = true
				return nil
			}
			if name == "main" {
				mainFound = true
				return nil
			}
			if name == "master" {
				masterFound = true
				return nil
			}
		}
		return nil
	})

	var tagFound bool
	if tag != "" {
		tagrefs, err := repo.Tags()
		if err != nil {
			return err
		}

		err = tagrefs.ForEach(func(t *plumbing.Reference) error {
			if tag == t.Name().Short() {
				tagFound = true
				r = t
				return nil
			}
			return nil
		})
	}

	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	if tagFound || branchFound {
		hash, err := repo.ResolveRevision(plumbing.Revision(r.Name().Short()))
		if err != nil {
			return err
		}
		err = w.Checkout(&git.CheckoutOptions{
			Hash: *hash,
		})
	} else {
		if mainFound {
			err = w.Checkout(&git.CheckoutOptions{
				Branch: plumbing.NewBranchReferenceName("main"),
			})
		} else if masterFound {
			err = w.Checkout(&git.CheckoutOptions{
				Branch: plumbing.NewBranchReferenceName("master"),
			})
		} else {
			return fmt.Errorf("No branch or tag found")
		}
	}

	return err
}
