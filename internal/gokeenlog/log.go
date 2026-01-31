// Package gokeenlog provides simple logging utilities for gokeenapi.
// It outputs formatted messages to stdout with consistent styling.
package gokeenlog

import (
	"fmt"
	"strings"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

// Info prints a message to stdout with a newline.
func Info(msg string) {
	fmt.Println(msg)
}

// HorizontalLine prints a visual separator line using emoji characters.
func HorizontalLine() {
	Info(strings.Repeat("➖", 5))
}

// Infof prints a formatted message to stdout with a newline.
func Infof(msg string, args ...any) {
	s := fmt.Sprintf(msg, args...)
	fmt.Printf("%v\n", s)
}

// InfoSubStepf prints a formatted sub-step message with a bullet point prefix.
// Used for displaying details under a main operation.
func InfoSubStepf(msg string, args ...any) {
	s := fmt.Sprintf(msg, args...)
	fmt.Printf("    ▪ %v\n", s)
}

// InfoSubStep prints a sub-step message with a bullet point prefix.
func InfoSubStep(msg string) {
	fmt.Printf("    ▪ %v\n", msg)
}

// PrintParseResponse prints parse response messages when debug logging is enabled.
// Used to display detailed API response information for troubleshooting.
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
