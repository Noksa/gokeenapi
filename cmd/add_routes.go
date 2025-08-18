package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/noksa/gokeenapi/internal/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAddRoutesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-routes",
		Aliases: []string{"addroutes"},
		Short:   "Add static routes in Keenetic router",
	}

	cmd.Flags().String("interface-id", "", "Keenetic interface ID to update routes on")
	cmd.Flags().StringSlice("bat-file", []string{}, "Path to a bat file to add routes from. Can be specified multiple times")
	cmd.Flags().StringSlice("bat-url", []string{}, "URL to a bat file to add routes from. Can be specified multiple times")

	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlag(config.ViperKeeneticInterfaceId, cmd.Flags().Lookup("interface-id"))
		_ = viper.BindPFlag(config.ViperBatFiles, cmd.Flags().Lookup("bat-file"))
		_ = viper.BindPFlag(config.ViperBatUrls, cmd.Flags().Lookup("bat-url"))
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(viper.GetStringSlice(config.ViperBatUrls)) == 0 && len(viper.GetStringSlice(config.ViperBatFiles)) == 0 {
			return fmt.Errorf("at least one of --bat-file or --bat-url must be set")
		}
		err := checkInterfaceId(viper.GetString(config.ViperKeeneticInterfaceId))
		if err != nil {
			return err
		}
		err = checkInterfaceExists(viper.GetString(config.ViperKeeneticInterfaceId))
		if err != nil {
			return err
		}
		for _, file := range viper.GetStringSlice(config.ViperBatFiles) {
			absFilePath, err := filepath.Abs(file)
			if err != nil {
				return err
			}
			err = gokeenrestapi.Route.AddRoutesFromBatFile(absFilePath)
			if err != nil {
				return err
			}
		}
		for _, url := range viper.GetStringSlice(config.ViperBatUrls) {
			err := gokeenrestapi.Route.AddRoutesFromBatUrl(url)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return cmd
}
