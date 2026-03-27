package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rodrigobatini/hamunaptra-cli/internal/api"
	"github.com/rodrigobatini/hamunaptra-cli/internal/configfile"
	"github.com/rodrigobatini/hamunaptra-cli/internal/localproj"
	"github.com/rodrigobatini/hamunaptra-cli/internal/providers/vercelcli"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Collect provider data locally and upload snapshots",
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

			conns, err := c.ListConnections(pid)
			if err != nil {
				return fmt.Errorf("failed to list project connections: %w", err)
			}

			var hasVercel bool
			for _, conn := range conns {
				if strings.EqualFold(conn.Provider, "vercel") {
					hasVercel = true
					break
				}
			}
			if !hasVercel {
				return fmt.Errorf("no vercel connection found; run `hamunaptra connect vercel`")
			}

			collector := vercelcli.NewCollector()
			snaps, err := collector.Collect(context.Background())
			if err != nil {
				msg := err.Error()
				if strings.Contains(strings.ToLower(msg), "not found") {
					msg += "\nInstall Vercel CLI: https://vercel.com/docs/cli"
				}
				if strings.Contains(strings.ToLower(msg), "session invalid") {
					msg += "\nRun: vercel login"
				}
				return fmt.Errorf(msg)
			}
			payload := api.PostSyncReq{Snapshots: make([]api.SyncSnapshot, 0, len(snaps))}
			for _, s := range snaps {
				payload.Snapshots = append(payload.Snapshots, api.SyncSnapshot{
					Date:      s.Date,
					Provider:  "vercel",
					AmountUSD: s.AmountUSD,
					Metadata:  s.Metadata,
				})
			}
			if err := c.PostSync(pid, payload); err != nil {
				return err
			}
			cmd.Printf("Sync complete. Uploaded %d snapshots.\n", len(payload.Snapshots))
			return nil
		},
	}
	cmd.Flags().StringVar(&projectID, "project", "", "project UUID")
	return cmd
}
