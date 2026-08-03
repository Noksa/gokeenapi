package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
)

func newDeleteAllDnsRoutingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdDeleteAllDnsRouting,
		Aliases: AliasesDeleteAllDnsRouting,
		Short:   "Remove all DNS-routing groups from the router",
		Long: `Delete all DNS-routing groups (object-groups and dns-proxy routes) from your Keenetic router.

This command fetches the current DNS-routing state directly from the router and removes
every group it finds, regardless of what is defined in your configuration file.

Use this command to fully reset DNS-routing before re-applying a new configuration,
for example after renaming groups.

Examples:
  # Delete all DNS-routing groups
  gokeenapi delete-all-dns-routing --config config.yaml

  # Delete without confirmation prompt
  gokeenapi delete-all-dns-routing --config config.yaml --force

Safety: This command deletes ALL DNS-routing groups. Use with caution.

Requirements: Keenetic firmware version 5.0.1 or higher`,
	}

	var force bool
	cmd.Flags().BoolVar(&force, "force", false,
		`Skip confirmation prompt and delete all DNS-routing groups immediately.
Use with caution as this bypasses the safety confirmation.`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if !force {
			confirmed, err := confirmAction(fmt.Sprintf("\n%v This will delete %v DNS-routing groups. Do you want to continue?",
				color.RedString("WARNING:"),
				color.CyanString("ALL")))
			if err != nil {
				return err
			}
			if !confirmed {
				gokeenlog.Info("Deletion cancelled")
				return nil
			}
		}

		return gokeenrestapi.DnsRouting.DeleteAllDnsRoutingGroups()
	}
	return cmd
}
