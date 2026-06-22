package cmd

import (
	"errors"
	"strings"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
)

func newUpdateAwgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdUpdateAwg,
		Aliases: AliasesUpdateAwg,
		Short:   "Update existing WireGuard VPN configuration",
		Long: `Update an existing WireGuard (AWG) connection with new configuration from a .conf file.

This command updates the configuration of an existing WireGuard interface using
a standard WireGuard configuration file. It compares current router state with
the conf file and applies only the changes.

Supports standard WireGuard, AmneziaWG 1.0 (Jc, Jmin, Jmax, S1, S2, H1-H4),
and AWG 2.0 (S3, S4, I1-I5) parameters. ASC fields are optional.

Use --dry-run to preview what would be changed without applying.

Examples:
  # Update existing WireGuard interface
  gokeenapi update-awg --config config.yaml --conf-file /path/to/updated.conf --interface-id Wireguard0

  # Preview changes without applying
  gokeenapi update-awg --config config.yaml --conf-file /path/to/updated.conf --interface-id Wireguard0 --dry-run`,
	}
	var confFile, interfaceId string
	var dryRun bool
	cmd.Flags().StringVar(&confFile, "conf-file", "", "Path to WireGuard configuration file (.conf)")
	cmd.Flags().StringVar(&interfaceId, "interface-id", "", "ID of the existing WireGuard interface to update")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be changed without applying")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if confFile == "" {
			return errors.New("--conf-file flag is required")
		}
		if interfaceId == "" {
			return errors.New("--interface-id flag is required")
		}
		if dryRun {
			diff, err := gokeenrestapi.AwgConf.DiffUpdate(confFile, interfaceId)
			if err != nil {
				return err
			}
			if diff == "" {
				gokeenlog.InfoSubStepf("Interface %v is already up to date — no changes needed", color.CyanString(interfaceId))
				return nil
			}
			gokeenlog.Infof("Changes for %v:", color.CyanString(interfaceId))
			printColoredDiff(diff)
			return nil
		}
		return gokeenrestapi.AwgConf.ConfigureOrUpdateInterface(confFile, interfaceId)
	}
	return cmd
}

// printColoredDiff prints a unified diff with colored output:
// red for removals, green for additions, cyan for hunk headers, bold for file headers.
func printColoredDiff(diff string) {
	for line := range strings.SplitSeq(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "---"), strings.HasPrefix(line, "+++"):
			gokeenlog.Info(color.New(color.Bold).Sprint(line))
		case strings.HasPrefix(line, "@@"):
			gokeenlog.Info(color.CyanString(line))
		case strings.HasPrefix(line, "-"):
			gokeenlog.Info(color.RedString(line))
		case strings.HasPrefix(line, "+"):
			gokeenlog.Info(color.GreenString(line))
		case line != "":
			gokeenlog.Info(line)
		}
	}
}
