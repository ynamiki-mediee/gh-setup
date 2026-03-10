package gh

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// DetectRepo returns the current repository in "owner/repo" format and its host
// by inspecting the git remote configuration via go-gh.
func DetectRepo() (nwo string, host string, err error) {
	repo, err := repository.Current()
	if err != nil {
		return "", "", fmt.Errorf("detecting repository: %w", err)
	}
	return fmt.Sprintf("%s/%s", repo.Owner, repo.Name), repo.Host, nil
}
