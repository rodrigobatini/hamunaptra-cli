package cli

import (
	"fmt"
	"os"

	"github.com/rodrigobatini/hamunaptra-cli/internal/tui"
	"github.com/spf13/cobra"
)

func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "hamunaptra",
		Short: "Hamunaptra — infra cost CLI + cloud backend",
		Long:  "Authenticate in the browser, link a project, connect providers, sync, report, and ask questions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Keep non-interactive usage script-friendly.
			if !isInteractiveTerminal() {
				return cmd.Help()
			}
			return tui.Run()
		},
	}
	root.AddCommand(
		newVersionCmd(),
		newLoginCmd(),
		newLogoutCmd(),
		newWhoamiCmd(),
		newInitCmd(),
		newConnectCmd(),
		newSyncCmd(),
		newReportCmd(),
		newAnomaliesCmd(),
		newAskCmd(),
		newDoctorCmd(),
	)
	return root
}

func isInteractiveTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(Version)
		},
	}
}
