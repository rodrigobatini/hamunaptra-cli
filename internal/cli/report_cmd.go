package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rodrigobatini/hamunaptra-cli/internal/api"
	"github.com/rodrigobatini/hamunaptra-cli/internal/configfile"
	"github.com/rodrigobatini/hamunaptra-cli/internal/localproj"
	"github.com/spf13/cobra"
)

func newReportCmd() *cobra.Command {
	var from, to, projectID string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Print cost report for the linked project",
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
			raw, err := c.GetReport(pid, from, to)
			if err != nil {
				return err
			}
			if jsonOut {
				cmd.Print(string(raw))
				return nil
			}
			var v any
			if err := json.Unmarshal(raw, &v); err != nil {
				return err
			}
			b, _ := json.MarshalIndent(v, "", "  ")
			cmd.Println(string(b))
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "start date YYYY-MM-DD")
	cmd.Flags().StringVar(&to, "to", "", "end date YYYY-MM-DD")
	cmd.Flags().StringVar(&projectID, "project", "", "project UUID")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "raw JSON")
	return cmd
}
