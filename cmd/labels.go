package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ynamiki-mediee/gh-setup/internal/config"
	"github.com/ynamiki-mediee/gh-setup/internal/gh"
	"github.com/ynamiki-mediee/gh-setup/internal/label"
	"github.com/ynamiki-mediee/gh-setup/internal/prompt"
)

var labelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "Sync GitHub labels from config",
	RunE:  runLabels,
}

func init() {
	rootCmd.AddCommand(labelsCmd)
}

func runLabels(cmd *cobra.Command, args []string) error {
	detected, host, err := gh.DetectRepo()
	if err != nil {
		return err
	}

	repo, err := prompt.ConfirmRepo(detected)
	if err != nil {
		if handleCancel(err) {
			return nil
		}
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if cfg == nil || len(cfg.Labels) == 0 {
		return fmt.Errorf("No labels defined in .gh-setup.yml")
	}

	client, err := gh.NewClient(host)
	if err != nil {
		return err
	}

	existing, err := client.ListLabels(repo)
	if err != nil {
		return err
	}

	desired := make([]label.Label, len(cfg.Labels))
	for i, l := range cfg.Labels {
		desired[i] = label.Label{
			Name:        l.Name,
			Color:       l.Color,
			Description: l.Description,
		}
	}

	diff := label.ComputeDiff(desired, existing)

	if len(diff.ToCreate) == 0 && len(diff.ToUpdate) == 0 {
		fmt.Println("All labels are up to date.")
		return nil
	}

	if len(diff.ToCreate) > 0 {
		fmt.Printf("Create (%d):\n", len(diff.ToCreate))
		for _, l := range diff.ToCreate {
			fmt.Printf("  + %s (#%s)\n", l.Name, strings.TrimPrefix(l.Color, "#"))
		}
	}
	if len(diff.ToUpdate) > 0 {
		fmt.Printf("Update (%d):\n", len(diff.ToUpdate))
		for _, l := range diff.ToUpdate {
			fmt.Printf("  ~ %s (#%s)\n", l.Name, strings.TrimPrefix(l.Color, "#"))
		}
	}
	fmt.Printf("Unchanged: %d\n", diff.Unchanged)

	ok, err := prompt.Confirm("Apply these changes?")
	if err != nil {
		if handleCancel(err) {
			return nil
		}
		return err
	}
	if !ok {
		fmt.Println("Cancelled.")
		return nil
	}

	var created, updated, failed int

	for _, l := range diff.ToCreate {
		if err := client.CreateLabel(repo, l.Name, l.Color, l.Description); err != nil {
			fmt.Printf("Failed to create %q: %v\n", l.Name, err)
			failed++
		} else {
			created++
		}
	}

	for _, l := range diff.ToUpdate {
		if err := client.UpdateLabel(repo, l.Name, l.Color, l.Description); err != nil {
			fmt.Printf("Failed to update %q: %v\n", l.Name, err)
			failed++
		} else {
			updated++
		}
	}

	fmt.Printf("Created: %d / Updated: %d / Failed: %d\n", created, updated, failed)

	if failed > 0 {
		return fmt.Errorf("%d label operation(s) failed", failed)
	}

	return nil
}
