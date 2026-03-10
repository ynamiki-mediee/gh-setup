package prompt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// ErrCancelled is returned when the user aborts a prompt.
var ErrCancelled = errors.New("cancelled by user")

// wrapErr converts huh.ErrUserAborted to ErrCancelled.
func wrapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrCancelled
	}
	return err
}

// TextInput prompts the user for a text value.
func TextInput(message, placeholder, initialValue string) (string, error) {
	var value string
	input := huh.NewInput().
		Title(message).
		Placeholder(placeholder).
		Value(&value)
	if initialValue != "" {
		value = initialValue
	}
	err := huh.NewForm(huh.NewGroup(input)).Run()
	if err != nil {
		return "", wrapErr(err)
	}
	return value, nil
}

// Confirm asks a yes/no question and returns the boolean choice.
func Confirm(message string) (bool, error) {
	var value bool
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(message).
			Value(&value),
	)).Run()
	if err != nil {
		return false, wrapErr(err)
	}
	return value, nil
}

// Select prompts the user to pick one option from a list.
func Select(message string, options []string) (string, error) {
	var value string
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(message).
			Options(opts...).
			Value(&value),
	)).Run()
	if err != nil {
		return "", wrapErr(err)
	}
	return value, nil
}

// SelectWithDefault prompts the user to pick one option, pre-selecting defaultVal.
func SelectWithDefault(message string, options []string, defaultVal string) (string, error) {
	value := defaultVal
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(message).
			Options(opts...).
			Value(&value),
	)).Run()
	if err != nil {
		return "", wrapErr(err)
	}
	return value, nil
}

// MultiSelect prompts the user to pick multiple options.
// defaults specifies which options are pre-selected.
func MultiSelect(message string, options []string, defaults []string) ([]string, error) {
	defaultSet := make(map[string]bool, len(defaults))
	for _, d := range defaults {
		defaultSet[d] = true
	}

	var selected []string
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opt := huh.NewOption(o, o)
		if defaultSet[o] {
			opt = opt.Selected(true)
		}
		opts[i] = opt
	}
	err := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title(message).
			Options(opts...).
			Value(&selected),
	)).Run()
	if err != nil {
		return nil, wrapErr(err)
	}
	return selected, nil
}

// ConfirmRepo asks the user to confirm the detected repository or enter one manually.
// If detected is empty, it asks for manual input directly.
func ConfirmRepo(detected string) (string, error) {
	if detected != "" {
		ok, err := Confirm(fmt.Sprintf("Use detected repository %q?", detected))
		if err != nil {
			return "", err
		}
		if ok {
			return detected, nil
		}
	}

	repo, err := TextInput("Enter repository (owner/repo)", "owner/repo", "")
	if err != nil {
		return "", err
	}
	repo = strings.TrimSpace(repo)
	if strings.Count(repo, "/") != 1 {
		return "", fmt.Errorf("invalid repository format %q: expected owner/repo", repo)
	}
	parts := strings.Split(repo, "/")
	if parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid repository format %q: owner and repo must not be empty", repo)
	}
	return repo, nil
}
