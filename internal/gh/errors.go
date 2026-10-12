package gh

import (
	"errors"
	"net/http"
	"strings"

	ghAPI "github.com/cli/go-gh/v2/pkg/api"
)

const planRestrictionHint = "Upgrade to GitHub"

// IsPlanRestricted reports whether err is a 403 saying the feature needs a paid plan.
func IsPlanRestricted(err error) bool {
	var httpErr *ghAPI.HTTPError
	if !errors.As(err, &httpErr) {
		return false
	}
	return httpErr.StatusCode == http.StatusForbidden &&
		strings.Contains(httpErr.Message, planRestrictionHint)
}
