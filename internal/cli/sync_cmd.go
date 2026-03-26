package cli

import (
	"fmt"
	"os"

	"github.com/rodrigobatini/hamunaptra-cli/internal/api"
	"github.com/rodrigobatini/hamunaptra-cli/internal/configfile"
	"github.com/rodrigobatini/hamunaptra-cli/internal/localproj"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Trigger backend sync (demo cost data in MVP)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configfile.Load()
			if err != nil {
				return err
			}
			base := cfg.APIBase
			if base == "" {
				base = configfile.DefaultAPIBase()
			}
			if cfg.AccessToken == "" {
				return fmt.Errorf("not logged in")
			}
			pid := projectID
			if pid == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				pid, err = localproj.ReadID(wd)
				if err != nil {
					return err
				}
			}
			c := api.New(base, cfg.AccessToken)
			if err := c.PostSync(pid); err != nil {
				return err
			}
			cmd.Println("Sync complete.")
			return nil
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project UUID")
	return cmd
}
