package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper writes content to .gh-setup.yml inside dir and chdirs there.
// It returns a cleanup function that restores the original working directory.
func setupConfigFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
}

func TestLoadConfig_ValidFull(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, `
milestones:
  startDate: "2025-01-06"
  weeks: 4
  timezone: "Asia/Tokyo"
labels:
  - name: bug
    color: "d73a4a"
    description: "Something isn't working"
  - name: feature
    color: "0075ca"
`)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Milestones == nil {
		t.Fatal("expected milestones")
	}
	if cfg.Milestones.StartDate != "2025-01-06" {
		t.Errorf("startDate = %q, want %q", cfg.Milestones.StartDate, "2025-01-06")
	}
	if cfg.Milestones.Weeks != 4 {
		t.Errorf("weeks = %d, want 4", cfg.Milestones.Weeks)
	}
	if cfg.Milestones.Timezone != "Asia/Tokyo" {
		t.Errorf("timezone = %q, want %q", cfg.Milestones.Timezone, "Asia/Tokyo")
	}
	if len(cfg.Labels) != 2 {
		t.Fatalf("labels count = %d, want 2", len(cfg.Labels))
	}
	if cfg.Labels[0].Name != "bug" {
		t.Errorf("labels[0].name = %q, want %q", cfg.Labels[0].Name, "bug")
	}
}

func TestLoadConfig_LabelsOnly(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, `
labels:
  - name: bug
    color: "d73a4a"
`)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Milestones != nil {
		t.Error("expected nil milestones")
	}
	if len(cfg.Labels) != 1 {
		t.Fatalf("labels count = %d, want 1", len(cfg.Labels))
	}
}

func TestLoadConfig_MilestonesOnly(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, `
milestones:
  startDate: "2025-01-06"
  weeks: 4
`)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Milestones == nil {
		t.Fatal("expected milestones")
	}
	if len(cfg.Labels) != 0 {
		t.Errorf("labels count = %d, want 0", len(cfg.Labels))
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config for missing file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, `{{{invalid yaml`)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing config file") {
		t.Errorf("error = %q, want it to contain %q", err, "parsing config file")
	}
}

func TestLoadConfig_LabelMissingName(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, `
labels:
  - color: "d73a4a"
`)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for label missing name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("error = %q, want it to contain %q", err, "name is required")
	}
}

func TestLoadConfig_LabelMissingColor(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, `
labels:
  - name: bug
`)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for label missing color")
	}
	if !strings.Contains(err.Error(), "color is required") {
		t.Errorf("error = %q, want it to contain %q", err, "color is required")
	}
}

func TestLoadConfig_MilestonesMissingStartDate(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, `
milestones:
  weeks: 4
`)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for milestones missing startDate")
	}
	if !strings.Contains(err.Error(), "startDate is required") {
		t.Errorf("error = %q, want it to contain %q", err, "startDate is required")
	}
}

func TestLoadConfig_MilestonesMissingWeeks(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, `
milestones:
  startDate: "2025-01-06"
`)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for milestones missing weeks")
	}
	if !strings.Contains(err.Error(), "weeks must be a positive integer") {
		t.Errorf("error = %q, want it to contain %q", err, "weeks must be a positive integer")
	}
}

func TestLoadConfig_InvalidStartDateFormat(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, `
milestones:
  startDate: "01-06-2025"
  weeks: 4
`)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid startDate format")
	}
	if !strings.Contains(err.Error(), "invalid startDate") {
		t.Errorf("error = %q, want it to contain %q", err, "invalid startDate")
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config for empty file")
	}
}

func TestLoadConfig_WhitespaceOnlyFile(t *testing.T) {
	dir := t.TempDir()
	setupConfigFile(t, dir, "  \n  \n")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config for whitespace-only file")
	}
}
