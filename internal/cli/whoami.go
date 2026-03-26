package cli

import (
	"encoding/json"
	"fmt"

	"github.com/rodrigobatini/hamunaptra-cli/internal/api"
	"github.com/rodrigobatini/hamunaptra-cli/internal/configfile"
	"github.com/spf13/cobra"
)

func newWhoamiCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show current API identity",
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
				return fmt.Errorf("not logged in; run hamunaptra login")
			}
			c := api.New(base, cfg.AccessToken)
			me, err := c.Me()
			if err != nil {
				return err
			}
			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(me)
			}
			cmd.Printf("user_id: %s\norg_id:  %s\nplan:    %s\n", me.UserID, me.OrgID, me.Plan)
			if me.Email != "" {
				cmd.Printf("email:   %s\n", me.Email)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print JSON")
	return cmd
}
