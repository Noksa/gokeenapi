package cmd

import (
	"errors"
	"fmt"
	"github.com/noksa/gokeenapi/internal/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/viper"
	"go.uber.org/multierr"
	"os"
	"runtime"
)

func checkRequiredFields() error {
	var mErr error
	if viper.GetString(config.ViperKeeneticUrl) == "" {
		mErr = multierr.Append(mErr, errors.New("please specify a keenetic url via flag/field/variable"))
	}
	if viper.GetString(config.ViperKeeneticLogin) == "" {
		mErr = multierr.Append(mErr, errors.New("please specify a keenetic login via flag/field/variable"))
	}
	if viper.GetString(config.ViperKeeneticPassword) == "" {
		mErr = multierr.Append(mErr, errors.New("please specify a keenetic password via flag/field/variable"))
	}

	return mErr
}

func checkInterfaceId() error {
	if viper.GetString(config.ViperKeeneticInterfaceId) == "" {
		return errors.New("please specify a keenetic interface id via flag/field/variable")
	}
	return nil
}

func checkInterfaceExists() error {
	interfaces, err := gokeenrestapi.Interface.GetInterfacesViaRciShowInterfaces()
	if err != nil {
		return err
	}
	interfaceFound := false
	for _, interfaceDetails := range interfaces {
		if interfaceDetails.Id == viper.GetString(config.ViperKeeneticInterfaceId) {
			interfaceFound = true
			break
		}
	}
	if !interfaceFound {
		return fmt.Errorf("keenetic router doesn't have interface with id '%v'. Verify that you specified correct ID", viper.GetString(config.ViperKeeneticInterfaceId))
	}
	return nil
}

func RestoreCursor() {
	if !(len(os.Getenv("WT_SESSION")) > 0 && runtime.GOOS == "windows") {
		// make sure to restore cursor in all cases
		_, _ = fmt.Fprint(os.Stdout, "\033[?25h")
	}
}
