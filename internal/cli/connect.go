package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rodrigobatini/hamunaptra-cli/internal/api"
	"github.com/rodrigobatini/hamunaptra-cli/internal/configfile"
	"github.com/rodrigobatini/hamunaptra-cli/internal/executil"
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
			provider := args[0]
			if provider == "vercel" {
				p, err := executil.LookPath("vercel")
				if err != nil {
					return fmt.Errorf("vercel cli not found on PATH; install vercel cli and run `vercel login`")
				}
				out, errOut, runErr := executil.Run(context.Background(), executil.DefaultTimeout, p, "whoami")
				if runErr != nil {
					if errOut == "" {
						errOut = out
					}
					return fmt.Errorf("vercel session not ready (%s); run `vercel login`", strings.TrimSpace(errOut))
				}
			}
			c := api.New(base, cfg.AccessToken)
			source := "manual"
			if provider == "vercel" {
				source = "vercel_cli"
			}
			if err := c.PostConnection(pid, provider, source); err != nil {
				return err
			}
			cmd.Println("Connected:", provider)
			return nil
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project UUID (default: hamunaptra.yaml)")
	return cmd
}
