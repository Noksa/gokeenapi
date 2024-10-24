package keenlog

import (
	"fmt"
	"github.com/noksa/gokeenapi/internal/config"
	"github.com/noksa/gokeenapi/pkg/models"
	"github.com/spf13/viper"
)

func Info(msg string) {
	fmt.Println(msg)
}

func Infof(msg string, args ...any) {
	s := fmt.Sprintf(msg, args...)
	fmt.Printf("%v\n", s)
}

func InfoSubStepf(msg string, args ...any) {
	s := fmt.Sprintf(msg, args...)
	fmt.Printf("      ▪ %v\n", s)
}

func Debug(msg string) {
	if viper.GetBool(config.ViperDebug) {
		fmt.Println(msg)
	}
}

func PrintParseResponse(parseResponse []models.ParseResponse) {
	if !viper.GetBool(config.ViperDebug) {
		return
	}
	if len(parseResponse) == 0 {
		return
	}
	for _, parse := range parseResponse {
		for _, status := range parse.Parse.Status {
			InfoSubStepf(status.Message)
		}
	}
}
