package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/spf13/cobra"
)

func newDeleteRoutesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdDeleteRoutes,
		Aliases: AliasesDeleteRoutes,
		Short:   "Remove routing rules from specified interfaces",
		Long: `Delete static routes from your Keenetic (Netcraze) router interfaces.

This command removes user-defined static routes from specified interfaces. By default,
it processes all interfaces defined in your configuration file. You can target a 
specific interface using the --interface-id flag.

The command will:
1. List all routes to be deleted
2. Ask for confirmation (unless --force is used)
3. Delete the confirmed routes

Examples:
  # Delete routes from all interfaces in config
  gokeenapi delete-routes --config config.yaml

  # Delete routes from specific interface
  gokeenapi delete-routes --config config.yaml --interface-id Wireguard0

  # Delete without confirmation prompt
  gokeenapi delete-routes --config config.yaml --force

Safety: Only user-defined static routes are deleted. System routes remain untouched.`,
	}

	var interfaceId string
	var force bool
	cmd.Flags().StringVar(&interfaceId, "interface-id", "",
		`Target a specific Keenetic (Netcraze) interface ID for route deletion.
If not specified, processes all interfaces from the config file.
Use 'show-interfaces' to list available interface IDs.`)
	cmd.Flags().BoolVar(&force, "force", false,
		`Skip confirmation prompt and delete routes immediately.
Use with caution as this bypasses the safety confirmation.`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var interfaces []string
		if interfaceId != "" {
			interfaces = append(interfaces, interfaceId)
		} else {
			for _, routeSetting := range config.Cfg.Routes {
				interfaces = append(interfaces, routeSetting.InterfaceID)
			}
		}

		type interfaceRoutes struct {
			interfaceId string
			routes      []gokeenrestapimodels.RciIpRoute
		}
		var allRoutesToDelete []interfaceRoutes
		var totalRoutes int

		for _, ifaceId := range interfaces {
			err := gokeenrestapi.Checks.CheckInterfaceId(ifaceId)
			if err != nil {
				return err
			}
			err = gokeenrestapi.Checks.CheckInterfaceExists(ifaceId)
			if err != nil {
				return err
			}
			routes, err := gokeenrestapi.Ip.GetAllUserRoutesRciIpRoute(ifaceId)
			if err != nil {
				return err
			}

			if len(routes) > 0 {
				totalRoutes += len(routes)
				allRoutesToDelete = append(allRoutesToDelete, interfaceRoutes{ifaceId, routes})
			}
		}

		if totalRoutes == 0 {
			gokeenlog.Info("No routes found to delete")
			return nil
		}

		for _, routeInfo := range allRoutesToDelete {
			for _, route := range routeInfo.routes {
				msg := ""
				if route.Host != "" {
					msg = color.CyanString(route.Host)
				} else {
					msg = color.CyanString(route.Network) + "/" + color.BlueString(route.Mask)
				}
				gokeenlog.InfoSubStepf("Route to delete: %v via %v",
					msg,
					color.YellowString(route.Interface))
			}
		}

		if !force {
			confirmed, err := confirmAction(fmt.Sprintf("\nFound %v total route(s) to delete. Do you want to continue?", color.CyanString("%v", totalRoutes)))
			if err != nil {
				return err
			}
			if !confirmed {
				gokeenlog.Info("Deletion cancelled")
				return nil
			}
		}

		for _, item := range allRoutesToDelete {
			err := gokeenrestapi.Ip.DeleteRoutes(item.routes, item.interfaceId)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return cmd
}
