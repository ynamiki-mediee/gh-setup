package label

import "testing"

func TestComputeDiff_AllNew(t *testing.T) {
	t.Parallel()
	desired := []Label{
		{Name: "bug", Color: "d73a4a", Description: "Bug report"},
		{Name: "feature", Color: "0075ca"},
	}
	result := ComputeDiff(desired, nil)

	if len(result.ToCreate) != 2 {
		t.Errorf("ToCreate = %d, want 2", len(result.ToCreate))
	}
	if len(result.ToUpdate) != 0 {
		t.Errorf("ToUpdate = %d, want 0", len(result.ToUpdate))
	}
	if result.Unchanged != 0 {
		t.Errorf("Unchanged = %d, want 0", result.Unchanged)
	}
}

func TestComputeDiff_AllUnchanged(t *testing.T) {
	t.Parallel()
	labels := []Label{
		{Name: "bug", Color: "d73a4a", Description: "Bug report"},
		{Name: "feature", Color: "0075ca"},
	}
	result := ComputeDiff(labels, labels)

	if len(result.ToCreate) != 0 {
		t.Errorf("ToCreate = %d, want 0", len(result.ToCreate))
	}
	if len(result.ToUpdate) != 0 {
		t.Errorf("ToUpdate = %d, want 0", len(result.ToUpdate))
	}
	if result.Unchanged != 2 {
		t.Errorf("Unchanged = %d, want 2", result.Unchanged)
	}
}

func TestComputeDiff_ColorChange(t *testing.T) {
	t.Parallel()
	desired := []Label{{Name: "bug", Color: "ff0000"}}
	existing := []Label{{Name: "bug", Color: "d73a4a"}}
	result := ComputeDiff(desired, existing)

	if len(result.ToUpdate) != 1 {
		t.Fatalf("ToUpdate = %d, want 1", len(result.ToUpdate))
	}
	if result.ToUpdate[0].Color != "ff0000" {
		t.Errorf("updated color = %q, want %q", result.ToUpdate[0].Color, "ff0000")
	}
}

func TestComputeDiff_DescriptionChange(t *testing.T) {
	t.Parallel()
	desired := []Label{{Name: "bug", Color: "d73a4a", Description: "Updated desc"}}
	existing := []Label{{Name: "bug", Color: "d73a4a", Description: "Old desc"}}
	result := ComputeDiff(desired, existing)

	if len(result.ToUpdate) != 1 {
		t.Fatalf("ToUpdate = %d, want 1", len(result.ToUpdate))
	}
	if result.ToUpdate[0].Description != "Updated desc" {
		t.Errorf("updated description = %q, want %q", result.ToUpdate[0].Description, "Updated desc")
	}
}

func TestComputeDiff_CaseInsensitiveMatch(t *testing.T) {
	t.Parallel()
	desired := []Label{{Name: "Bug", Color: "d73a4a"}}
	existing := []Label{{Name: "bug", Color: "d73a4a"}}
	result := ComputeDiff(desired, existing)

	if len(result.ToCreate) != 0 {
		t.Errorf("ToCreate = %d, want 0", len(result.ToCreate))
	}
	if result.Unchanged != 1 {
		t.Errorf("Unchanged = %d, want 1", result.Unchanged)
	}
}

func TestComputeDiff_HashPrefixStripped(t *testing.T) {
	t.Parallel()
	desired := []Label{{Name: "bug", Color: "#d73a4a"}}
	existing := []Label{{Name: "bug", Color: "d73a4a"}}
	result := ComputeDiff(desired, existing)

	if len(result.ToUpdate) != 0 {
		t.Errorf("ToUpdate = %d, want 0 (hash prefix should be ignored)", len(result.ToUpdate))
	}
	if result.Unchanged != 1 {
		t.Errorf("Unchanged = %d, want 1", result.Unchanged)
	}
}

func TestComputeDiff_ColorCaseInsensitive(t *testing.T) {
	t.Parallel()
	desired := []Label{{Name: "bug", Color: "D73A4A"}}
	existing := []Label{{Name: "bug", Color: "d73a4a"}}
	result := ComputeDiff(desired, existing)

	if len(result.ToUpdate) != 0 {
		t.Errorf("ToUpdate = %d, want 0 (color comparison should be case-insensitive)", len(result.ToUpdate))
	}
	if result.Unchanged != 1 {
		t.Errorf("Unchanged = %d, want 1", result.Unchanged)
	}
}

func TestComputeDiff_EmptyDesired(t *testing.T) {
	t.Parallel()
	existing := []Label{
		{Name: "bug", Color: "d73a4a"},
		{Name: "feature", Color: "0075ca"},
	}
	result := ComputeDiff(nil, existing)

	if len(result.ToCreate) != 0 {
		t.Errorf("ToCreate = %d, want 0", len(result.ToCreate))
	}
	if len(result.ToUpdate) != 0 {
		t.Errorf("ToUpdate = %d, want 0", len(result.ToUpdate))
	}
	if result.Unchanged != 0 {
		t.Errorf("Unchanged = %d, want 0", result.Unchanged)
	}
}

func TestComputeDiff_Mixed(t *testing.T) {
	t.Parallel()
	desired := []Label{
		{Name: "bug", Color: "d73a4a", Description: "Bug report"},       // unchanged
		{Name: "feature", Color: "ff0000"},                               // update (color differs)
		{Name: "docs", Color: "0075ca", Description: "Documentation"},    // create (new)
	}
	existing := []Label{
		{Name: "bug", Color: "d73a4a", Description: "Bug report"},
		{Name: "Feature", Color: "0075ca"},
	}
	result := ComputeDiff(desired, existing)

	if len(result.ToCreate) != 1 {
		t.Errorf("ToCreate = %d, want 1", len(result.ToCreate))
	}
	if len(result.ToUpdate) != 1 {
		t.Errorf("ToUpdate = %d, want 1", len(result.ToUpdate))
	}
	if result.Unchanged != 1 {
		t.Errorf("Unchanged = %d, want 1", result.Unchanged)
	}

	if len(result.ToCreate) == 1 && result.ToCreate[0].Name != "docs" {
		t.Errorf("ToCreate[0].Name = %q, want %q", result.ToCreate[0].Name, "docs")
	}
	if len(result.ToUpdate) == 1 && result.ToUpdate[0].Name != "feature" {
		t.Errorf("ToUpdate[0].Name = %q, want %q", result.ToUpdate[0].Name, "feature")
	}
}
