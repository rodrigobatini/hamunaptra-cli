package cli

import (
	"context"
	"fmt"

	"github.com/rodrigobatini/hamunaptra-cli/internal/executil"
	"github.com/spf13/cobra"
)

type toolCheck struct {
	Provider string
	Label    string
	Bin      string
	Args     []string
}

var doctorTools = []toolCheck{
	{Provider: "vercel", Label: "Vercel CLI", Bin: "vercel", Args: []string{"--version"}},
	{Provider: "supabase", Label: "Supabase CLI", Bin: "supabase", Args: []string{"--version"}},
	{Provider: "neon", Label: "Neon CLI", Bin: "neonctl", Args: []string{"--version"}},
	{Provider: "render", Label: "Render CLI", Bin: "render", Args: []string{"--version"}},
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check official provider CLIs on PATH",
		Long:  "Runs each known provider CLI with a short --version (or equivalent) and reports OK or missing.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var missing int
			for _, t := range doctorTools {
				path, err := executil.LookPath(t.Bin)
				if err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-12s  %s  not found on PATH\n", t.Provider, t.Label)
					missing++
					continue
				}
				out, errOut, runErr := executil.Run(ctx, executil.DefaultTimeout, path, t.Args...)
				line := out
				if line == "" && errOut != "" {
					line = errOut
				}
				if runErr != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-12s  %s  error: %v", t.Provider, t.Label, runErr)
					if line != "" {
						fmt.Fprintf(cmd.OutOrStdout(), " (%s)", line)
					}
					fmt.Fprintln(cmd.OutOrStdout())
					missing++
					continue
				}
				if len(line) > 80 {
					line = line[:77] + "..."
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-12s  %s  ok  %s\n", t.Provider, t.Label, line)
			}
			if missing > 0 {
				return fmt.Errorf("%d tool(s) missing or failing — install provider CLIs and add them to PATH", missing)
			}
			return nil
		},
	}
}
