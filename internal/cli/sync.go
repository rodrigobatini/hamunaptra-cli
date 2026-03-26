package cli

import "github.com/spf13/cobra"

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync usage data from providers (stub)",
		Long:  "Planned: orchestrate provider CLIs and aggregate results. Not implemented in this MVP.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("sync: not implemented yet — see roadmap; the previous cloud flow lives in hamunaptra-api branch legacy/web-dashboard.")
			return nil
		},
	}
}
