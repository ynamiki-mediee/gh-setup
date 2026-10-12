package gh

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	ghAPI "github.com/cli/go-gh/v2/pkg/api"
)

func TestIsPlanRestricted(t *testing.T) {
	const planMsg = "Upgrade to GitHub Pro or make this repository public to enable this feature."

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("boom"), false},
		{"403 plan restriction", &ghAPI.HTTPError{StatusCode: http.StatusForbidden, Message: planMsg}, true},
		{"wrapped 403 plan restriction", fmt.Errorf("wrap: %w", &ghAPI.HTTPError{StatusCode: http.StatusForbidden, Message: planMsg}), true},
		{"403 without hint", &ghAPI.HTTPError{StatusCode: http.StatusForbidden, Message: "Resource not accessible by integration"}, false},
		{"404 with hint", &ghAPI.HTTPError{StatusCode: http.StatusNotFound, Message: planMsg}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPlanRestricted(tt.err); got != tt.want {
				t.Errorf("IsPlanRestricted() = %v, want %v", got, tt.want)
			}
		})
	}
}
