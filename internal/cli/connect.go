package cli

import (
	"fmt"
	"os"

	"github.com/rodrigobatini/hamunaptra-cli/internal/api"
	"github.com/rodrigobatini/hamunaptra-cli/internal/configfile"
	"github.com/rodrigobatini/hamunaptra-cli/internal/localproj"
	"github.com/spf13/cobra"
)

func newConnectCmd() *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "connect [provider]",
		Short: "Register a provider connection for the linked project",
		Args:  cobra.ExactArgs(1),
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
					return fmt.Errorf("no project linked; run hamunaptra init or pass --project")
				}
			}
			c := api.New(base, cfg.AccessToken)
			if err := c.PostConnection(pid, args[0]); err != nil {
				return err
			}
			cmd.Println("Connected:", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project UUID (default: hamunaptra.yaml)")
	return cmd
}
