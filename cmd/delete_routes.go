package cmd

import (
	"github.com/noksa/gokeenapi/internal/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newDeleteRoutesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete-routes",
		Aliases: []string{"deleteroutes"},
		Short:   "Delete static routes in Keenetic router",
	}

	cmd.Flags().String("interface-id", "", "Keenetic interface ID to delete static routes on")

	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlag(config.ViperKeeneticInterfaceId, cmd.Flags().Lookup("interface-id"))
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := checkInterfaceId()
		if err != nil {
			return err
		}
		err = checkInterfaceExists()
		if err != nil {
			return err
		}
		routes, err := gokeenrestapi.Route.GetAllUserRoutesRciIpRoute(viper.GetString(config.ViperKeeneticInterfaceId))
		if err != nil {
			return err
		}
		return gokeenrestapi.Route.DeleteRoutes(routes)
	}
	return cmd
}
