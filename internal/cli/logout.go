package cli

import (
	"github.com/rodrigobatini/hamunaptra-cli/internal/configfile"
	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove local CLI credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configfile.Load()
			if err != nil {
				return err
			}
			cfg.AccessToken = ""
			return configfile.Save(cfg)
		},
	}
}
