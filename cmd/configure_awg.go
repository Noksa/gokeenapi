package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/config"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/keenspinner"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
)

func newConfigureAwgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "configure-awg",
		Aliases: []string{"configureawg"},
		Short:   "Configure Wireguard connection to add ASC parameters to it in Keenetic router",
	}

	var confPath string
	cmd.Flags().String("interface-id", "", "Keenetic interface ID to configure")
	cmd.Flags().StringVar(&confPath, "conf-file", "", "Path to a conf TOML file with AWG configuration")

	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlag(config.ViperKeeneticInterfaceId, cmd.Flags().Lookup("interface-id"))
		_ = viper.BindPFlag(config.ViperKeeneticInterfaceConfFile, cmd.Flags().Lookup("conf-file"))
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if confPath == "" {
			return fmt.Errorf("conf-file flag is required")
		}
		err := checkInterfaceId()
		if err != nil {
			return err
		}
		err = checkInterfaceExists()
		if err != nil {
			return err
		}
		var Jcstring, Jminstring, Jmaxstring, S1string, S2string, H1string, H2string, H3string, H4string string
		confPath, err = filepath.Abs(confPath)
		if err != nil {
			return err
		}
		gokeenlog.InfoSubStepf("Conf-file: %v", color.CyanString("%v", confPath))
		err = keenspinner.WrapWithSpinner("Reading AWG config file", func() error {
			cfg, err := ini.Load(confPath)
			if err != nil {
				return err
			}
			interfaceSection, err := cfg.GetSection("Interface")
			if err != nil {
				return err
			}
			Jc, err := interfaceSection.GetKey("Jc")
			if err != nil {
				return err
			}
			Jcstring = Jc.String()
			Jmin, err := interfaceSection.GetKey("Jmin")
			if err != nil {
				return err
			}
			Jminstring = Jmin.String()
			Jmax, err := interfaceSection.GetKey("Jmax")
			if err != nil {
				return err
			}
			Jmaxstring = Jmax.String()
			S1, err := interfaceSection.GetKey("S1")
			if err != nil {
				return err
			}
			S1string = S1.String()
			S2, err := interfaceSection.GetKey("S2")
			if err != nil {
				return err
			}
			S2string = S2.String()
			H1, err := interfaceSection.GetKey("H1")
			if err != nil {
				return err
			}
			H1string = H1.String()
			H2, err := interfaceSection.GetKey("H2")
			if err != nil {
				return err
			}
			H2string = H2.String()
			H3, err := interfaceSection.GetKey("H3")
			if err != nil {
				return err
			}
			H3string = H3.String()
			H4, err := interfaceSection.GetKey("H4")
			if err != nil {
				return err
			}
			H4string = H4.String()
			return nil
		})
		if err != nil {
			return err
		}
		return gokeenrestapi.AwgConf.ConfigureOrUpdateInterface(Jcstring, Jminstring, Jmaxstring, S1string, S2string, H1string, H2string, H3string, H4string)
	}
	return cmd
}
