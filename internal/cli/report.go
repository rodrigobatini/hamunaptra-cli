package cli

import "github.com/spf13/cobra"

func newReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Print an aggregated cost report (stub)",
		Long:  "Planned: summarize last sync. Not implemented in this MVP.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("report: not implemented yet — run `hamunaptra doctor` to verify provider CLIs.")
			return nil
		},
	}
}
