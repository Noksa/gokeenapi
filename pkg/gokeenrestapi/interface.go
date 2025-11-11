package gokeenrestapi

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeencache"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

type keeneticInterface struct {
}

var (
	// Interface provides network interface management functionality for the router
	Interface keeneticInterface
)

// GetInterfaceViaRciShowInterfaces retrieves detailed information about a specific interface
func (*keeneticInterface) GetInterfaceViaRciShowInterfaces(interfaceId string) (gokeenrestapimodels.RciShowInterface, error) {
	var myInterface gokeenrestapimodels.RciShowInterface
	body, err := Common.ExecuteGetSubPath(fmt.Sprintf("/rci/show/interface/%v", interfaceId))
	if err != nil {
		return myInterface, err
	}
	err = json.Unmarshal(body, &myInterface)
	return myInterface, err
}

// GetInterfaceViaRciShowScInterfaces retrieves system configuration details for a specific interface
func (*keeneticInterface) GetInterfaceViaRciShowScInterfaces(interfaceId string) (gokeenrestapimodels.RciShowScInterface, error) {
	var myInterface gokeenrestapimodels.RciShowScInterface
	body, err := Common.ExecuteGetSubPath(fmt.Sprintf("/rci/show/sc/interface/%v", interfaceId))
	if err != nil {
		return myInterface, err
	}
	err = json.Unmarshal(body, &myInterface)
	return myInterface, err
}

// GetInterfacesViaRciShowInterfaces retrieves all interfaces with optional type filtering and caching
func (*keeneticInterface) GetInterfacesViaRciShowInterfaces(useCache bool, interfaceTypes ...string) (map[string]gokeenrestapimodels.RciShowInterface, error) {
	var interfaces map[string]gokeenrestapimodels.RciShowInterface
	if useCache {
		interfaces = gokeencache.GetRciShowInterfaces()
	}
	if interfaces == nil {
		err := gokeenspinner.WrapWithSpinner(fmt.Sprintf("Fetching %v", color.CyanString("interfaces")), func() error {
			body, err := Common.ExecuteGetSubPath("/rci/show/interface")
			if err != nil {
				return err
			}
			return json.Unmarshal(body, &interfaces)
		})
		if err != nil {
			return interfaces, err
		}
		gokeencache.SetRciShowInterfaces(interfaces)
	}
	if len(interfaceTypes) == 0 {
		return interfaces, nil
	}
	realInterfaces := map[string]gokeenrestapimodels.RciShowInterface{}
	for k, interfaceDetails := range interfaces {
		for _, v := range interfaceTypes {
			if strings.EqualFold(interfaceDetails.Type, v) {
				realInterfaces[k] = interfaceDetails
			}
		}
	}
	return realInterfaces, nil
}

// GetInterfacesViaRciShowScInterfaces retrieves system configuration for all or specified interfaces
func (*keeneticInterface) GetInterfacesViaRciShowScInterfaces(ids ...string) (map[string]gokeenrestapimodels.RciShowScInterface, error) {
	var interfaces map[string]gokeenrestapimodels.RciShowScInterface
	err := gokeenspinner.WrapWithSpinner(fmt.Sprintf("Fetching %v", color.CyanString("interfaces")), func() error {
		body, err := Common.ExecuteGetSubPath("/rci/show/sc/interface")
		if err != nil {
			return err
		}
		return json.Unmarshal(body, &interfaces)
	})
	if err != nil {
		return interfaces, err
	}
	if len(ids) == 0 {
		return interfaces, nil
	}
	realInterfaces := map[string]gokeenrestapimodels.RciShowScInterface{}
	for k, interfaceDetails := range interfaces {
		if !slices.Contains(ids, k) {
			continue
		}
		realInterfaces[k] = interfaceDetails
	}
	return realInterfaces, nil
}

// PrintInfoAboutInterfaces displays formatted information about interfaces to the console
func (*keeneticInterface) PrintInfoAboutInterfaces(interfaces map[string]gokeenrestapimodels.RciShowInterface) {
	for k, interfaceDetails := range interfaces {
		gokeenlog.Infof("Interface '%v':", color.BlueString(k))
		gokeenlog.InfoSubStepf("Id: %v", color.CyanString(interfaceDetails.Id))
		gokeenlog.InfoSubStepf("Type: %v", color.CyanString(interfaceDetails.Type))
		if interfaceDetails.Description != "" {
			gokeenlog.InfoSubStepf("Description: %v", color.CyanString(interfaceDetails.Description))
		}
		if interfaceDetails.Address != "" {
			gokeenlog.InfoSubStepf("Address: %v", color.CyanString(interfaceDetails.Address))
		}
		gokeenlog.Infof("")
	}

}

// WaitUntilInterfaceIsUp waits up to 60 seconds for an interface to become fully operational
func (*keeneticInterface) WaitUntilInterfaceIsUp(interfaceId string) error {
	err := gokeenspinner.WrapWithSpinner(fmt.Sprintf("Waiting 60s until %v interface is up, connected to peers and working", interfaceId), func() error {
		deadline := time.Now().Add(time.Second * 60)
		for time.Now().Before(deadline) {
			myInterface, err := Interface.GetInterfaceViaRciShowInterfaces(interfaceId)
			if err != nil {
				return err
			}
			if myInterface.Connected == StateConnected && myInterface.Link == StateUp && myInterface.State == StateUp {
				return nil
			}
			time.Sleep(time.Millisecond * 500)
		}
		return fmt.Errorf("looks like interface %v is still not up. Please check The keenetic web-interface", interfaceId)
	})
	return err
}

// UpInterface brings the specified interface up (enables it)
func (*keeneticInterface) UpInterface(interfaceId string) error {
	var parseSlice []gokeenrestapimodels.ParseRequest
	parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
		Parse: fmt.Sprintf("interface %v up", interfaceId),
	})
	var parseResponse []gokeenrestapimodels.ParseResponse
	err := gokeenspinner.WrapWithSpinner(fmt.Sprintf("Bringing %v interface up", color.CyanString(interfaceId)), func() error {
		var executeErr error
		parseSlice = Common.EnsureSaveConfigAtEnd(parseSlice)
		parseResponse, executeErr = Common.ExecutePostParse(parseSlice...)
		return executeErr
	})
	gokeenlog.PrintParseResponse(parseResponse)
	return err
}

// SetGlobalIpInInterface configures global IP routing for the specified interface
func (*keeneticInterface) SetGlobalIpInInterface(interfaceId string, global bool) error {
	var parseSlice []gokeenrestapimodels.ParseRequest
	val := "ip global auto"
	if !global {
		val = "no ip global"
	}
	parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
		Parse: fmt.Sprintf("interface %v %v", interfaceId, val),
	})
	var parseResponse []gokeenrestapimodels.ParseResponse
	err := gokeenspinner.WrapWithSpinner(fmt.Sprintf("Changing global IP in %v interface to %v", color.CyanString(interfaceId), color.GreenString("%v", global)), func() error {
		var executeErr error
		parseSlice = Common.EnsureSaveConfigAtEnd(parseSlice)
		parseResponse, executeErr = Common.ExecutePostParse(parseSlice...)
		return executeErr
	})
	gokeenlog.PrintParseResponse(parseResponse)
	return err
}
