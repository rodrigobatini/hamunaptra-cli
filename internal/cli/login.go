package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/rodrigobatini/hamunaptra-cli/internal/api"
	"github.com/rodrigobatini/hamunaptra-cli/internal/configfile"
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	var apiBase string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate via browser (device flow)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configfile.Load()
			if err != nil {
				return err
			}
			base := apiBase
			if base == "" {
				base = cfg.APIBase
			}
			if base == "" {
				base = configfile.DefaultAPIBase()
			}
			c := api.New(base, "")
			start, err := c.CLIStart()
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
				p, err := c.CLIPoll(start.DeviceCode)
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
					return nil
				case "complete":
					return fmt.Errorf("session already consumed; run login again")
				case "pending":
					continue
				default:
					continue
				}
			}
		},
	}
	cmd.Flags().StringVar(&apiBase, "api", "", "API base URL (default: config or HAMUNAPTRA_API)")
	return cmd
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
