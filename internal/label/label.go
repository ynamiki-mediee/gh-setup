package label

import "strings"

// Label represents a GitHub label.
type Label struct {
	Name        string
	Color       string
	Description string
}

// DiffResult holds the outcome of comparing desired labels against existing ones.
type DiffResult struct {
	ToCreate  []Label
	ToUpdate  []Label
	Unchanged int
}

// ComputeDiff compares desired labels against existing labels and returns
// which labels need to be created, updated, or are unchanged.
//
// Matching is case-insensitive on the label name. Color comparison strips any
// leading '#' prefix and is also case-insensitive.
func ComputeDiff(desired []Label, existing []Label) DiffResult {
	existingMap := make(map[string]Label, len(existing))
	for _, l := range existing {
		existingMap[strings.ToLower(l.Name)] = l
	}

	var result DiffResult
	for _, d := range desired {
		key := strings.ToLower(d.Name)
		e, found := existingMap[key]
		if !found {
			result.ToCreate = append(result.ToCreate, d)
			continue
		}

		if labelsEqual(d, e) {
			result.Unchanged++
		} else {
			result.ToUpdate = append(result.ToUpdate, d)
		}
	}

	return result
}

// normalizeColor strips a leading '#' and lowercases the color string.
func normalizeColor(c string) string {
	return strings.ToLower(strings.TrimPrefix(c, "#"))
}

func labelsEqual(a, b Label) bool {
	if normalizeColor(a.Color) != normalizeColor(b.Color) {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	return true
}
