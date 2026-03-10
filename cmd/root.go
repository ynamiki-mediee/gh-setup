package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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
	rootCmd.SetVersionTemplate(fmt.Sprintf("gh-setup version %s\n", version))
}

func Execute() error {
	return rootCmd.Execute()
}
