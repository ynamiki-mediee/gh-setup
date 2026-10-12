package gh

import (
	"errors"
	"net/http"
	"strings"

	ghAPI "github.com/cli/go-gh/v2/pkg/api"
)

// planRestrictionHint is the phrase GitHub includes in the 403 message when a
// feature requires a paid plan, e.g. "Upgrade to GitHub Pro or make this
// repository public to enable this feature."
const planRestrictionHint = "Upgrade to GitHub"

// IsPlanRestricted reports whether err is a GitHub API error saying the
// feature is unavailable on the repository's current plan.
func IsPlanRestricted(err error) bool {
	var httpErr *ghAPI.HTTPError
	if !errors.As(err, &httpErr) {
		return false
	}
	return httpErr.StatusCode == http.StatusForbidden &&
		strings.Contains(httpErr.Message, planRestrictionHint)
}
