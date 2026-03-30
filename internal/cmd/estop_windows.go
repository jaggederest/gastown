//go:build windows

// Windows stub for gt estop / gt thaw — SIGTSTP/SIGCONT are not available on Windows.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/estop"
	"github.com/steveyegge/gastown/internal/style"
)

var estopCmd = &cobra.Command{
	Use:     "estop",
	GroupID: GroupServices,
	Short:   "Emergency stop (not supported on Windows)",
	Long:    "gt estop requires POSIX signals (SIGTSTP/SIGCONT) which are not available on Windows.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("gt estop is not supported on Windows")
	},
}

var thawCmd = &cobra.Command{
	Use:     "thaw",
	GroupID: GroupServices,
	Short:   "Resume from emergency stop (not supported on Windows)",
	Long:    "gt thaw requires POSIX signals (SIGCONT) which are not available on Windows.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("gt thaw is not supported on Windows")
	},
}

func init() {
	rootCmd.AddCommand(estopCmd)
	rootCmd.AddCommand(thawCmd)
}

// addEstopToStatus checks for E-stop and prints a banner if active.
func addEstopToStatus(townRoot string) {
	if estop.IsActive(townRoot) {
		fmt.Printf("%s  E-STOP ACTIVE\n\n", style.Error.Render("⛔"))
	}
}
