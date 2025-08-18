package gokeenrestapi

import (
	"fmt"

	"github.com/noksa/gokeenapi/internal/config"
	"github.com/noksa/gokeenapi/internal/keenspinner"
	"github.com/noksa/gokeenapi/pkg/models"
	"github.com/spf13/viper"
)

var AwgConf keeneticAwgconf

type keeneticAwgconf struct{}

func (*keeneticAwgconf) ConfigureOrUpdateInterface(jc, jmin, jmax, s1, s2, h1, h2, h3, h4 string) error {
	var parseSlice []models.ParseRequest
	keeneticInterface := viper.GetString(config.ViperKeeneticInterfaceId)
	confParse := models.ParseRequest{}
	confParse.Parse = fmt.Sprintf("interface %v wireguard asc %v %v %v %v %v %v %v %v %v",
		keeneticInterface,
		jc,
		jmin,
		jmax,
		s1,
		s2,
		h1,
		h2,
		h3,
		h4)
	parseSlice = append(parseSlice, confParse,
		models.ParseRequest{Parse: "system configuration save"})
	return keenspinner.WrapWithSpinner(fmt.Sprintf("Configuring %v interface with ASC parameters", keeneticInterface), func() error {
		_, err := ExecutePostParse(parseSlice...)
		return err
	})
}
