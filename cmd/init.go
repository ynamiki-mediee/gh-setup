package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/ynamiki-mediee/gh-setup/internal/gh"
	"github.com/ynamiki-mediee/gh-setup/internal/prompt"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive repository setup wizard",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// contains reports whether s is present in the slice.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// task represents a single apply operation with a title and function.
type task struct {
	title string
	fn    func() error
}

func runInit(cmd *cobra.Command, args []string) error {
	// --- Step 1: Repo detection ---
	detected, err := gh.DetectRepo()
	if err != nil {
		return err
	}

	repo, err := prompt.ConfirmRepo(detected)
	if err != nil {
		if errors.Is(err, prompt.ErrCancelled) {
			fmt.Println("Setup cancelled.")
			return nil
		}
		return err
	}
	fmt.Printf("Repository: %s\n\n", repo)

	// --- Step 2: Branch protection ---
	client, err := gh.NewClient()
	if err != nil {
		return err
	}

	defaultBranch := "main"
	if db, _, getErr := client.GetRepoInfo(repo); getErr == nil && db != "" {
		defaultBranch = db
	}

	branch, err := prompt.TextInput("Branch to protect", "main", defaultBranch)
	if err != nil {
		if errors.Is(err, prompt.ErrCancelled) {
			fmt.Println("Setup cancelled.")
			return nil
		}
		return err
	}

	branchProtectionOptions := []string{
		"Block direct pushes",
		"Require PR reviews",
		"Require status checks",
		"Require conversation resolution",
		"Enforce for admins",
		"Allow force pushes",
		"Block branch deletion",
	}
	branchProtectionDefaults := []string{
		"Block direct pushes",
		"Block branch deletion",
	}

	branchRules, err := prompt.MultiSelect("Branch protection rules", branchProtectionOptions, branchProtectionDefaults)
	if err != nil {
		if errors.Is(err, prompt.ErrCancelled) {
			fmt.Println("Setup cancelled.")
			return nil
		}
		return err
	}

	requiredApprovals := 0
	if contains(branchRules, "Require PR reviews") {
		approvalStr, err := prompt.Select("Required approvals", []string{"1", "2", "3"})
		if err != nil {
			if errors.Is(err, prompt.ErrCancelled) {
				fmt.Println("Setup cancelled.")
				return nil
			}
			return err
		}
		requiredApprovals, _ = strconv.Atoi(approvalStr)
	}

	branchProtection := gh.BranchProtectionSettings{
		BlockDirectPushes:             contains(branchRules, "Block direct pushes"),
		RequirePrReviews:              contains(branchRules, "Require PR reviews"),
		RequiredApprovals:             requiredApprovals,
		RequireStatusChecks:           contains(branchRules, "Require status checks"),
		RequireConversationResolution: contains(branchRules, "Require conversation resolution"),
		EnforceAdmins:                 contains(branchRules, "Enforce for admins"),
		AllowForcePushes:              contains(branchRules, "Allow force pushes"),
		BlockDeletion:                 contains(branchRules, "Block branch deletion"),
	}

	// --- Step 3: Repository settings ---
	repoSettingsOptions := []string{
		"Auto-delete branches after merge",
		"Allow squash merge",
		"Allow merge commit",
		"Allow rebase merge",
	}
	repoSettingsDefaults := []string{
		"Auto-delete branches after merge",
		"Allow squash merge",
	}

	repoSettingsSelected, err := prompt.MultiSelect("Repository settings", repoSettingsOptions, repoSettingsDefaults)
	if err != nil {
		if errors.Is(err, prompt.ErrCancelled) {
			fmt.Println("Setup cancelled.")
			return nil
		}
		return err
	}

	// Check if any merge strategy is selected.
	hasMergeStrategy := contains(repoSettingsSelected, "Allow squash merge") ||
		contains(repoSettingsSelected, "Allow merge commit") ||
		contains(repoSettingsSelected, "Allow rebase merge")

	if !hasMergeStrategy {
		fmt.Println("No merge strategy selected \u2014 PRs cannot be merged.")
		ok, err := prompt.Confirm("Continue anyway?")
		if err != nil {
			if errors.Is(err, prompt.ErrCancelled) {
				fmt.Println("Setup cancelled.")
				return nil
			}
			return err
		}
		if !ok {
			fmt.Println("Setup cancelled.")
			return nil
		}
	}

	repoSettings := gh.RepoSettings{
		DeleteBranchOnMerge: contains(repoSettingsSelected, "Auto-delete branches after merge"),
		AllowSquashMerge:    contains(repoSettingsSelected, "Allow squash merge"),
		AllowMergeCommit:    contains(repoSettingsSelected, "Allow merge commit"),
		AllowRebaseMerge:    contains(repoSettingsSelected, "Allow rebase merge"),
	}

	// --- Step 4: Security ---
	securityOptions := []string{
		"Dependabot alerts",
		"Dependabot security updates",
		"Secret scanning",
		"Secret scanning push protection",
	}
	securityDefaults := []string{
		"Dependabot alerts",
	}

	securitySelected, err := prompt.MultiSelect("Security features", securityOptions, securityDefaults)
	if err != nil {
		if errors.Is(err, prompt.ErrCancelled) {
			fmt.Println("Setup cancelled.")
			return nil
		}
		return err
	}

	security := gh.SecuritySettings{
		DependabotAlerts:             contains(securitySelected, "Dependabot alerts"),
		DependabotSecurityUpdates:    contains(securitySelected, "Dependabot security updates"),
		SecretScanning:               contains(securitySelected, "Secret scanning"),
		SecretScanningPushProtection: contains(securitySelected, "Secret scanning push protection"),
	}

	// --- Step 5: Summary + Confirmation ---
	fmt.Println()
	fmt.Println("Settings to apply:")
	fmt.Printf("Repository: %s\n", repo)

	fmt.Printf("\nBranch protection (%s):\n", branch)
	for _, rule := range branchRules {
		fmt.Printf("  + %s\n", rule)
	}
	if contains(branchRules, "Require PR reviews") {
		fmt.Printf("  + Required approvals: %d\n", requiredApprovals)
	}

	fmt.Println("\nRepository settings:")
	for _, s := range repoSettingsSelected {
		fmt.Printf("  + %s\n", s)
	}

	fmt.Println("\nSecurity:")
	for _, s := range securitySelected {
		fmt.Printf("  + %s\n", s)
	}
	fmt.Println()

	ok, err := prompt.Confirm("Apply these settings?")
	if err != nil {
		if errors.Is(err, prompt.ErrCancelled) {
			fmt.Println("Setup cancelled.")
			return nil
		}
		return err
	}
	if !ok {
		fmt.Println("Setup cancelled.")
		return nil
	}

	// --- Step 6: Apply ---
	var tasks []task

	tasks = append(tasks, task{
		title: "Branch protection",
		fn: func() error {
			return client.UpdateBranchProtection(repo, branch, branchProtection)
		},
	})

	tasks = append(tasks, task{
		title: "Repository settings",
		fn: func() error {
			return client.UpdateRepoSettings(repo, repoSettings)
		},
	})

	if security.DependabotAlerts {
		tasks = append(tasks, task{
			title: "Dependabot alerts",
			fn:    func() error { return client.EnableDependabotAlerts(repo) },
		})
	}
	if security.DependabotSecurityUpdates {
		tasks = append(tasks, task{
			title: "Dependabot security updates",
			fn:    func() error { return client.EnableDependabotSecurityUpdates(repo) },
		})
	}
	if security.SecretScanning {
		tasks = append(tasks, task{
			title: "Secret scanning",
			fn:    func() error { return client.EnableSecretScanning(repo) },
		})
	}
	if security.SecretScanningPushProtection {
		tasks = append(tasks, task{
			title: "Secret scanning push protection",
			fn:    func() error { return client.EnableSecretScanningPushProtection(repo) },
		})
	}

	var succeeded, failed int
	for _, t := range tasks {
		fmt.Printf("Applying: %s...\n", t.title)
		if err := t.fn(); err != nil {
			fmt.Printf("\u2717 %s: %v\n", t.title, err)
			failed++
		} else {
			fmt.Printf("\u2713 %s\n", t.title)
			succeeded++
		}
	}

	fmt.Printf("\nDone: %d succeeded, %d failed.\n", succeeded, failed)

	if failed > 0 {
		return fmt.Errorf("%d task(s) failed", failed)
	}

	return nil
}
