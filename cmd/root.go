package cmd

import (
	"github.com/joho/godotenv"
	"github.com/noksa/gokeenapi/internal/config"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenversion"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
	"os"
	"strings"
)
import "github.com/spf13/viper"

func NewRootCmd() *cobra.Command {
	viper.Reset()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.MustBindEnv(config.ViperKeeneticUrl, "GOKEENAPI_URL")
	viper.MustBindEnv(config.ViperKeeneticLogin, "GOKEENAPI_LOGIN")
	viper.MustBindEnv(config.ViperKeeneticPassword, "GOKEENAPI_PASSWORD")
	viper.MustBindEnv(config.ViperDebug, "GOKEENAPI_DEBUG")
	viper.MustBindEnv(config.ViperKeeneticInterfaceId, "GOKEENAPI_INTERFACE")
	viper.MustBindEnv(config.ViperKeeneticConfig, "GOKEENAPI_CONFIG")
	rootCmd := &cobra.Command{}
	rootCmd.SilenceUsage = true
	var configFile string
	rootCmd.Use = "gokeenrestapi"
	rootCmd.Short = "A convenient utility to add/delete routes in Keenetic routers via REST API"
	rootCmd.PersistentFlags().String("url", "", "Keenetic router url/ip address, example: http://192.168.1.1")
	rootCmd.PersistentFlags().String("login", "", "Keenetic API login")
	rootCmd.PersistentFlags().String("password", "", "Keenetic API password")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode and logging")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to YAML config file (optional)")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		_ = viper.BindPFlag(config.ViperDebug, rootCmd.PersistentFlags().Lookup("debug"))
		_ = viper.BindPFlag(config.ViperKeeneticUrl, rootCmd.PersistentFlags().Lookup("url"))
		_ = viper.BindPFlag(config.ViperKeeneticLogin, rootCmd.PersistentFlags().Lookup("login"))
		_ = viper.BindPFlag(config.ViperKeeneticPassword, rootCmd.PersistentFlags().Lookup("password"))
		_ = viper.BindPFlag(config.ViperKeeneticConfig, rootCmd.PersistentFlags().Lookup("config"))
		if configFile != "" {
			viper.SetConfigFile(configFile)
			viper.SetConfigType("yaml")
			err := viper.ReadInConfig()
			if err != nil {
				return err
			}
		}
		_, statErr := os.Stat(".gokeenapienv")
		if statErr == nil {
			err := godotenv.Load(".gokeenapienv")
			if err != nil {
				return err
			}
		}
		err := checkRequiredFields()
		if err != nil {
			return err
		}
		gokeenlog.Infof("Version: %v, BuildDate: %v", gokeenversion.Version(), gokeenversion.BuildDate())
		gokeenlog.Info("Configuration loaded:")
		gokeenlog.InfoSubStepf("Keenetic url: %v", viper.GetString(config.ViperKeeneticUrl))
		if viper.GetString(config.ViperKeeneticConfig) != "" {
			gokeenlog.InfoSubStepf("Config: %v", viper.GetString(config.ViperKeeneticConfig))
		}
		return gokeenrestapi.Auth()
	}

	rootCmd.AddCommand(
		newAddRoutesCmd(),
		newDeleteRoutesCmd(),
		newShowInterfacesCmd(),
	)
	return rootCmd
}
