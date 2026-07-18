package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
)

func newDeleteAllRoutesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdDeleteAllRoutes,
		Aliases: AliasesDeleteAllRoutes,
		Short:   "Remove all static routes from the router",
		Long: `Delete all static routes from your Keenetic (Netcraze) router in a single request.

This command sends a single REST API call that removes every user-defined static
route on the router at once. No route listing or per-route deletion is performed.

Examples:
  # Delete all routes
  gokeenapi delete-all-routes --config config.yaml

  # Delete without confirmation prompt
  gokeenapi delete-all-routes --config config.yaml --force

Safety: This command deletes ALL static routes. Use with caution.`,
	}

	var force bool
	cmd.Flags().BoolVar(&force, "force", false,
		`Skip confirmation prompt and delete routes immediately.
Use with caution as this bypasses the safety confirmation.`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if !force {
			confirmed, err := confirmAction(fmt.Sprintf("\n%v This will delete %v static routes. Do you want to continue?", color.RedString("WARNING:"), color.CyanString("ALL")))
			if err != nil {
				return err
			}
			if !confirmed {
				gokeenlog.Info("Deletion cancelled")
				return nil
			}
		}

		return gokeenrestapi.Ip.DeleteAllRoutes()
	}
	return cmd
}
