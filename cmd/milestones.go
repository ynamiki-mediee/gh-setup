package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/ynamiki-mediee/gh-setup/internal/config"
	"github.com/ynamiki-mediee/gh-setup/internal/gh"
	"github.com/ynamiki-mediee/gh-setup/internal/milestone"
	"github.com/ynamiki-mediee/gh-setup/internal/prompt"
)

var milestonesCmd = &cobra.Command{
	Use:   "milestones",
	Short: "Create or update weekly milestones",
	RunE:  runMilestones,
}

func init() {
	rootCmd.AddCommand(milestonesCmd)
}

var timezoneOptions = []struct{ label, value string }{
	{"Asia/Tokyo (JST, UTC+9)", "Asia/Tokyo"},
	{"Asia/Shanghai (CST, UTC+8)", "Asia/Shanghai"},
	{"Asia/Kolkata (IST, UTC+5:30)", "Asia/Kolkata"},
	{"Europe/Berlin (CET, UTC+1)", "Europe/Berlin"},
	{"Europe/London (GMT, UTC+0)", "Europe/London"},
	{"America/New_York (EST, UTC-5)", "America/New_York"},
	{"America/Chicago (CST, UTC-6)", "America/Chicago"},
	{"America/Los_Angeles (PST, UTC-8)", "America/Los_Angeles"},
	{"UTC", "UTC"},
}

func runMilestones(cmd *cobra.Command, args []string) error {
	// Step 1: Detect and confirm repo
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

	// Step 2-3: Get milestone parameters from config or interactive prompts
	var startDate string
	var weeks int
	var timezone string

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	if cfg != nil && cfg.Milestones != nil {
		startDate = cfg.Milestones.StartDate
		weeks = cfg.Milestones.Weeks
		timezone = cfg.Milestones.Timezone
		if timezone == "" {
			timezone = "UTC"
		}
	} else {
		// Interactive prompts
		startDate, err = prompt.TextInput("Start date (YYYY-MM-DD):", "", milestone.NextMonday(time.Now()))
		if err != nil {
			if handleCancel(err) {
				return nil
			}
			return err
		}

		weeksUntilEnd, err := milestone.WeeksUntilEndOfYear(startDate)
		if err != nil {
			return err
		}
		weeksDefault := strconv.Itoa(weeksUntilEnd)
		weeksStr, err := prompt.TextInput("Number of weeks:", "", weeksDefault)
		if err != nil {
			if handleCancel(err) {
				return nil
			}
			return err
		}
		weeks, err = strconv.Atoi(weeksStr)
		if err != nil {
			return fmt.Errorf("invalid number of weeks %q: %w", weeksStr, err)
		}

		// Build timezone select
		labels := make([]string, len(timezoneOptions))
		for i, o := range timezoneOptions {
			labels[i] = o.label
		}

		// Auto-detect system timezone
		systemTZ := time.Now().Location().String()
		defaultLabel := "UTC" // last option label
		for _, o := range timezoneOptions {
			if o.value == systemTZ {
				defaultLabel = o.label
				break
			}
		}

		selected, err := prompt.SelectWithDefault("Timezone for due dates:", labels, defaultLabel)
		if err != nil {
			if handleCancel(err) {
				return nil
			}
			return err
		}

		// Map selected label back to value
		for _, o := range timezoneOptions {
			if o.label == selected {
				timezone = o.value
				break
			}
		}
	}

	// Step 4: Create client and fetch existing milestones
	client, err := gh.NewClient(host)
	if err != nil {
		return err
	}

	fmt.Println("Fetching existing milestones...")
	existing, err := client.ListMilestones(repo)
	if err != nil {
		return err
	}

	// Step 5: Build existing map: title → milestone number
	existingMap := make(map[string]int)
	for _, m := range existing {
		existingMap[m.Title] = m.Number
	}

	// Step 6: Confirm
	ok, err := prompt.Confirm(fmt.Sprintf("Create/update %d weekly milestones starting from %s?", weeks, startDate))
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

	// Step 7: Loop and create/update milestones
	baseDate, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("invalid start date %q: %w", startDate, err)
	}

	var created, updated, failed int
	var failures []string

	for i := 0; i < weeks; i++ {
		weekStart := baseDate.AddDate(0, 0, i*7)
		weekEnd := weekStart.AddDate(0, 0, 6)

		weekNum := milestone.ISOWeek(weekStart)
		startDateStr := weekStart.Format("2006-01-02")
		endDateStr := weekEnd.Format("2006-01-02")

		title := fmt.Sprintf("Week %d: %s", weekNum, endDateStr)
		description := fmt.Sprintf("Period: %s - %s", startDateStr, endDateStr)

		dueOn, err := milestone.ToUtcDueOn(endDateStr, timezone)
		if err != nil {
			msg := fmt.Sprintf("Failed to compute due_on for %s: %v", endDateStr, err)
			fmt.Println(msg)
			failures = append(failures, msg)
			failed++
			continue
		}

		if num, exists := existingMap[title]; exists {
			// Update existing milestone
			fmt.Printf("Updating: %s...\n", title)
			if err := client.UpdateMilestone(repo, num, title, description, dueOn); err != nil {
				fmt.Printf("✗ %s: %v\n", title, err)
				failures = append(failures, fmt.Sprintf("Failed to update milestone %q: %v", title, err))
				failed++
			} else {
				fmt.Printf("✓ %s\n", title)
				updated++
			}
		} else {
			// Create new milestone
			fmt.Printf("Creating: %s...\n", title)
			if err := client.CreateMilestone(repo, title, description, dueOn); err != nil {
				fmt.Printf("✗ %s: %v\n", title, err)
				failures = append(failures, fmt.Sprintf("Failed to create milestone %q: %v", title, err))
				failed++
			} else {
				fmt.Printf("✓ %s\n", title)
				created++
			}
		}
	}

	// Step 8: Print summary
	if len(failures) > 0 {
		fmt.Println("\nFailures:")
		for _, f := range failures {
			fmt.Printf("  - %s\n", f)
		}
	}

	fmt.Printf("Created: %d / Updated: %d / Failed: %d\n", created, updated, failed)

	if failed > 0 {
		return fmt.Errorf("%d milestone operation(s) failed", failed)
	}

	return nil
}
