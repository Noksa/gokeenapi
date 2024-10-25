package gokeenrestapi

import (
	"encoding/json"
	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/keenspinner"
	"github.com/noksa/gokeenapi/pkg/models"
	"strings"
)

type keeneticInterface struct {
}

var Interface keeneticInterface

func (*keeneticInterface) GetInterfacesViaRciShowInterfaces(interfaceTypes ...string) (map[string]models.RciShowInterface, error) {
	var interfaces map[string]models.RciShowInterface
	err := keenspinner.WrapWithSpinner("Fetching interfaces", func() error {
		body, err := ExecuteGetSubPath("/rci/show/interface")
		if err != nil {
			return err
		}
		return json.Unmarshal(body, &interfaces)
	})
	if err != nil {
		return interfaces, err
	}
	if len(interfaceTypes) == 0 {
		return interfaces, nil
	}
	realInterfaces := map[string]models.RciShowInterface{}
	for k, interfaceDetails := range interfaces {
		k := k
		interfaceDetails := interfaceDetails
		for _, v := range interfaceTypes {
			v := v
			if strings.EqualFold(interfaceDetails.Type, v) {
				realInterfaces[k] = interfaceDetails
			}
		}
	}
	return realInterfaces, nil
}

func (*keeneticInterface) PrintInfoAboutInterfaces(interfaces map[string]models.RciShowInterface) {
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
