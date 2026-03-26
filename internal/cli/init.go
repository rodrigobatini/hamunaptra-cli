package cli

import (
	"fmt"
	"os"

	"github.com/rodrigobatini/hamunaptra-cli/internal/api"
	"github.com/rodrigobatini/hamunaptra-cli/internal/configfile"
	"github.com/rodrigobatini/hamunaptra-cli/internal/localproj"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var name, slug, dir string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a cloud project and link hamunaptra.yaml in the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
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
			if dir == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				dir = wd
			}
			c := api.New(base, cfg.AccessToken)
			id, err := c.CreateProject(api.ProjectCreate{Name: name, Slug: slug})
			if err != nil {
				return err
			}
			if err := localproj.WriteID(dir, id); err != nil {
				return err
			}
			cmd.Printf("Linked project %s → %s/%s\n", id, dir, localproj.FileName)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "project display name")
	cmd.Flags().StringVar(&slug, "slug", "", "url slug (optional)")
	cmd.Flags().StringVar(&dir, "dir", "", "directory for hamunaptra.yaml (default: cwd)")
	return cmd
}
