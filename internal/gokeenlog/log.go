package gokeenlog

import (
	"fmt"
	"strings"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

func Info(msg string) {
	fmt.Println(msg)
}

func HorizontalLine() {
	Info(strings.Repeat("➖", 5))
}

func Infof(msg string, args ...any) {
	s := fmt.Sprintf(msg, args...)
	fmt.Printf("%v\n", s)
}

func InfoSubStepf(msg string, args ...any) {
	s := fmt.Sprintf(msg, args...)
	fmt.Printf("    ▪ %v\n", s)
}

func InfoSubStep(msg string) {
	fmt.Printf("    ▪ %v\n", msg)
}

func PrintParseResponse(parseResponse []gokeenrestapimodels.ParseResponse) {
	if !config.Cfg.Logs.Debug {
		return
	}
	if len(parseResponse) == 0 {
		return
	}
	for _, parse := range parseResponse {
		for _, status := range parse.Parse.Status {
			InfoSubStep(status.Message)
		}
	}
}
