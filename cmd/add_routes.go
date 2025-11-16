package cmd

import (
	"path/filepath"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
)

func newAddRoutesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdAddRoutes,
		Aliases: AliasesAddRoutes,
		Short:   "Add routing rules from .bat files and URLs",
		Long: `Add static routes to your Keenetic (Netcraze) router from .bat files and remote URLs.

This command processes route definitions from local .bat files and remote URLs specified 
in your configuration file. Routes are added to the interfaces defined in the 'routes' 
section of your config.

The .bat files should contain Windows-style route commands:
  route add <network> mask <netmask> <gateway>

Examples:
  # Add all routes from config file
  gokeenapi add-routes --config config.yaml

  # Routes will be added to interfaces specified in config:
  # - Local .bat files are processed first
  # - Remote .bat URLs are downloaded and processed
  # - Routes are validated before being added to the router

Note: Use 'show-interfaces' command to verify interface IDs before adding routes.`,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		for _, addRouteSettings := range config.Cfg.Routes {
			err := gokeenrestapi.Checks.CheckInterfaceId(addRouteSettings.InterfaceID)
			if err != nil {
				return err
			}
			err = gokeenrestapi.Checks.CheckInterfaceExists(addRouteSettings.InterfaceID)
			if err != nil {
				return err
			}
			for _, file := range addRouteSettings.BatFile {
				absFilePath, err := filepath.Abs(file)
				if err != nil {
					return err
				}
				err = gokeenrestapi.Ip.AddRoutesFromBatFile(absFilePath, addRouteSettings.InterfaceID)
				if err != nil {
					return err
				}
			}
			for _, url := range addRouteSettings.BatURL {
				err := gokeenrestapi.Ip.AddRoutesFromBatUrl(url, addRouteSettings.InterfaceID)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	return cmd
}
