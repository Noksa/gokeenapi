package gokeenrestapi

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/go-resty/resty/v2"
	"github.com/noksa/gokeenapi/internal/config"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/keenspinner"
	"github.com/noksa/gokeenapi/pkg/models"
	"github.com/spf13/viper"
	"go.uber.org/multierr"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	regex = `(?i)route ADD (\d+.\d+.\d+.\d+) MASK (\d+.\d+.\d+.\d+)`
)

type keeneticRoute struct {
}

var Route keeneticRoute

func (*keeneticRoute) GetAllUserRoutesRciIpRoute(keeneticInterface string) ([]models.RciIpRoute, error) {
	var routes []models.RciIpRoute
	err := keenspinner.WrapWithSpinner("Fetching static routes", func() error {
		body, err := ExecuteGetSubPath("/rci/ip/route")
		if err != nil {
			return err
		}
		return json.Unmarshal(body, &routes)
	})
	if err != nil {
		return nil, err
	}
	var realRoutes []models.RciIpRoute
	for _, route := range routes {
		route := route
		if route.Interface == keeneticInterface {
			realRoutes = append(realRoutes, route)
		}
	}
	gokeenlog.InfoSubStepf("Found %v static routes for %v interface", color.MagentaString("%v", len(realRoutes)), viper.GetString(config.ViperKeeneticInterfaceId))
	return realRoutes, err
}

func (*keeneticRoute) DeleteRoutes(routes []models.RciIpRoute) error {
	if len(routes) == 0 {
		gokeenlog.Info("No need to delete static routes")
		return nil
	}
	var parseSlice []models.ParseRequest
	keeneticInterface := viper.GetString(config.ViperKeeneticInterfaceId)
	for _, route := range routes {
		if route.Interface != keeneticInterface {
			continue
		}
		parse := models.ParseRequest{}
		var ip string
		if route.Host != "" {
			ip = route.Host
		}
		if route.Network != "" {
			ip = fmt.Sprintf("%s %s", route.Network, route.Mask)
		}
		parse.Parse = fmt.Sprintf("no ip route %v %v", ip, keeneticInterface)
		parseSlice = append(parseSlice, parse)
	}
	return keenspinner.WrapWithSpinner(fmt.Sprintf("Deleting %v static routes with %v interface", color.MagentaString("%v", len(parseSlice)), keeneticInterface), func() error {
		_, err := ExecutePostParse(parseSlice...)
		return err
	})
}

func (*keeneticRoute) AddRoutesFromBatFile(batFile string) error {
	matcher := regexp.MustCompile(regex)
	b, err := os.ReadFile(batFile)
	if err != nil {
		return err
	}
	str := string(b)
	var mErr error
	splitted := strings.Split(str, "\n")
	var parseSlice []models.ParseRequest
	for _, line := range splitted {
		if line == "" {
			continue
		}
		sl := matcher.FindStringSubmatch(line)
		if len(sl) != 3 {
			gokeenlog.Infof("Skipping line with invalid format: '%v'", line)
			gokeenlog.InfoSubStepf("It doesn't satisfy regexp: '%v'", regex)
			mErr = multierr.Append(mErr, fmt.Errorf("line has invalid format: '%v'", line))
			continue
		}
		ip := sl[1]
		mask := sl[2]
		parseSlice = append(parseSlice, models.ParseRequest{Parse: fmt.Sprintf("ip route %v %v %v auto", ip, mask, viper.GetString(config.ViperKeeneticInterfaceId))})
	}
	var parseResponse []models.ParseResponse
	mErr = multierr.Append(mErr, keenspinner.WrapWithSpinner(fmt.Sprintf("Adding %v static routes from %v file to %v interface", color.MagentaString("%v", len(parseSlice)), color.CyanString(batFile), color.BlueString(viper.GetString(config.ViperKeeneticInterfaceId))), func() error {
		var executeErr error
		parseResponse, executeErr = ExecutePostParse(parseSlice...)
		return executeErr
	}))
	gokeenlog.PrintParseResponse(parseResponse)
	return mErr
}

func (*keeneticRoute) AddRoutesFromBatUrl(url string) error {
	matcher := regexp.MustCompile(regex)
	rClient := resty.New()
	rClient.SetDisableWarn(true)
	rClient.SetTimeout(time.Second * 5)
	var err error
	var response *resty.Response
	err = keenspinner.WrapWithSpinner(fmt.Sprintf("Fetching %v url", color.CyanString(url)), func() error {
		response, err = rClient.R().Get(url)
		return err
	})
	if err != nil {
		return err
	}
	str := string(response.Body())
	var mErr error
	splitted := strings.Split(str, "\n")
	var parseSlice []models.ParseRequest
	for _, line := range splitted {
		if line == "" {
			continue
		}
		sl := matcher.FindStringSubmatch(line)
		if len(sl) != 3 {
			gokeenlog.Infof("Skipping line with invalid format: '%v'", line)
			gokeenlog.InfoSubStepf("It doesn't satisfy regexp: '%v'", regex)
			mErr = multierr.Append(mErr, fmt.Errorf("line has invalid format: '%v'", line))
			continue
		}
		ip := sl[1]
		mask := sl[2]
		parseSlice = append(parseSlice, models.ParseRequest{Parse: fmt.Sprintf("ip route %v %v %v auto", ip, mask, viper.GetString(config.ViperKeeneticInterfaceId))})
	}
	var parseResponse []models.ParseResponse
	mErr = multierr.Append(mErr, keenspinner.WrapWithSpinner(fmt.Sprintf("Adding %v static routes to %v interface", color.MagentaString("%v", len(parseSlice)), color.BlueString(viper.GetString(config.ViperKeeneticInterfaceId))), func() error {
		var executeErr error
		parseResponse, executeErr = ExecutePostParse(parseSlice...)
		return executeErr
	}))
	gokeenlog.PrintParseResponse(parseResponse)
	return mErr
}
