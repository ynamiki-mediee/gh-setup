package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ynamiki-mediee/gh-setup/internal/prompt"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "gh-setup",
	Short: "Interactive GitHub repository setup CLI",
	Long:  "gh-setup — branch protection, milestones, labels & more.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("gh-setup version {{.Version}}\n")
	rootCmd.SilenceUsage = true
}

func Execute() error {
	return rootCmd.Execute()
}

// handleCancel checks if err is a user cancellation. If so, it prints
// a message and returns true, signaling the caller to return nil.
func handleCancel(err error) bool {
	if errors.Is(err, prompt.ErrCancelled) {
		fmt.Println("Cancelled.")
		return true
	}
	return false
}
