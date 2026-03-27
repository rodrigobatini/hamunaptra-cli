package cli

import (
	"fmt"
	"os"
	"time"

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
				// Avoid a login/init loop: when there is no local token, run device login inline.
				cmd.Println("You are not logged in. Starting browser login...")
				auth := api.New(base, "")
				start, err := auth.CLIStart()
				if err != nil {
					return err
				}
				cmd.Println("Open this URL in your browser to authorize the CLI:")
				cmd.Println(start.VerificationURL)
				cmd.Println("User code:", start.UserCode)
				_ = openBrowser(start.VerificationURL)

				interval := time.Duration(start.Interval) * time.Second
				if interval < time.Second {
					interval = 3 * time.Second
				}
				for {
					time.Sleep(interval)
					p, err := auth.CLIPoll(start.DeviceCode)
					if err != nil {
						return err
					}
					switch p.Status {
					case "authorized":
						cfg.APIBase = base
						cfg.AccessToken = p.AccessToken
						if err := configfile.Save(cfg); err != nil {
							return err
						}
						cmd.Println("Logged in.")
						goto loggedIn
					case "complete":
						return fmt.Errorf("session already consumed; run hamunaptra login")
					case "pending":
						continue
					default:
						continue
					}
				}
			}
		loggedIn:
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
