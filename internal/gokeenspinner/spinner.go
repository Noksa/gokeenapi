// Package gokeenspinner provides terminal spinner utilities for long-running operations.
// It displays progress indicators with timing information and supports both
// interactive terminals (with animated spinners) and non-interactive environments.
package gokeenspinner

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"golang.org/x/term"
)

// SpinnerOptions provides configuration for spinner behavior and post-completion actions.
type SpinnerOptions struct {
	mutex               sync.Mutex
	actionsAfterSpinner []func()
}

// AddActionAfterSpinner registers a function to be called after the spinner completes.
// Multiple actions can be registered and will be executed in order.
// This is useful for printing additional information after the operation finishes.
func (opts *SpinnerOptions) AddActionAfterSpinner(fn func()) {
	opts.mutex.Lock()
	defer opts.mutex.Unlock()
	opts.actionsAfterSpinner = append(opts.actionsAfterSpinner, fn)
}

// WrapWithSpinner wraps a function with a spinner display.
// This is a simplified version that doesn't expose SpinnerOptions.
// Use WrapWithSpinnerAndOptions for more control over post-completion behavior.
func WrapWithSpinner(spinnerText string, f func() error) error {
	return WrapWithSpinnerAndOptions(spinnerText, func(opts *SpinnerOptions) error {
		return f()
	})
}

// WrapWithSpinnerAndOptions wraps a function with a spinner and provides options for customization.
// In interactive terminals, displays an animated spinner with elapsed time.
// In non-interactive environments (CI, pipes), prints simple status messages.
// The spinnerText is displayed alongside the spinner/status.
func WrapWithSpinnerAndOptions(spinnerText string, f func(*SpinnerOptions) error) error {
	opts := &SpinnerOptions{}
	startTime := time.Now()

	// Check if we're in an interactive terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		// Non-interactive: just print start message and run function
		fmt.Printf("⌛   %v ...\n", spinnerText)
		err := f(opts)
		duration := getPrettyFormatedDuration(time.Since(startTime).Round(time.Millisecond))
		if err != nil {
			fmt.Printf("⛔   %v failed after %v\n", spinnerText, duration)
		} else {
			fmt.Printf("✅   %v completed after %v\n", spinnerText, duration)
		}
		for _, action := range opts.actionsAfterSpinner {
			action()
		}
		gokeenlog.HorizontalLine()
		return err
	}

	// Interactive terminal: use spinner
	s := spinner.New(spinner.CharSets[70], 100*time.Millisecond)
	s.Start()
	s.PostUpdate = func(s *spinner.Spinner) {
		s.Prefix = fmt.Sprintf("⌛   %v ... %s	", spinnerText, getPrettyFormatedDuration(time.Since(startTime).Round(time.Millisecond)))
	}
	s.Prefix = fmt.Sprintf("⌛   %v ...", spinnerText)
	err := f(opts)
	s.Prefix = spinnerText
	if err != nil {
		s.FinalMSG = fmt.Sprintf("⛔   %v failed after %v\n", spinnerText, getPrettyFormatedDuration(time.Since(startTime).Round(time.Millisecond)))
	} else {
		s.FinalMSG = fmt.Sprintf("✅   %v completed after %v\n", spinnerText, getPrettyFormatedDuration(time.Since(startTime).Round(time.Millisecond)))
	}
	s.Stop()

	for _, action := range opts.actionsAfterSpinner {
		action()
	}
	gokeenlog.HorizontalLine()
	return err
}

func getPrettyFormatedDuration(dur time.Duration) string {
	val := ""
	minute := int(dur.Minutes())
	second := int(dur.Seconds())
	if minute > 0 {
		second = second - (60 * minute)
	}
	if second == 0 {
		return dur.Round(time.Millisecond).String()
	}
	ms := fmt.Sprintf("%v", dur.Milliseconds())
	ms = ms[len(ms)-3:]
	if minute > 0 {
		val = fmt.Sprintf("%vm", minute)
	}
	if second > 0 {
		val = fmt.Sprintf("%v%v", val, second)
	}
	return fmt.Sprintf("%v.%vs", val, ms)
}
