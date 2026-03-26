package cli

import (
	"fmt"
	"os"

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
		Short: "Hamunaptra — orchestrate provider CLIs for cost visibility",
		Long:  "Local CLI that wraps official tools (Vercel, Supabase, Neon, Render, …) to aggregate usage and cost signals.",
	}
	root.AddCommand(
		newVersionCmd(),
		newDoctorCmd(),
		newSyncCmd(),
		newReportCmd(),
	)
	return root
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
