package gokeenspinner

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"golang.org/x/term"
)

func WrapWithSpinner(spinnerText string, f func() error) error {
	startTime := time.Now()

	// Check if we're in an interactive terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		// Non-interactive: just print start message and run function
		fmt.Printf("⌛   %v ...\n", spinnerText)
		err := f()
		duration := getPrettyFormatedDuration(time.Since(startTime).Round(time.Millisecond))
		if err != nil {
			fmt.Printf("⛔   %v failed after %v\n", spinnerText, duration)
		} else {
			fmt.Printf("✅   %v completed after %v\n", spinnerText, duration)
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
	err := f()
	s.Prefix = spinnerText
	if err != nil {
		s.FinalMSG = fmt.Sprintf("⛔   %v failed after %v\n", spinnerText, getPrettyFormatedDuration(time.Since(startTime).Round(time.Millisecond)))
	} else {
		s.FinalMSG = fmt.Sprintf("✅   %v completed after %v\n", spinnerText, getPrettyFormatedDuration(time.Since(startTime).Round(time.Millisecond)))
	}
	s.Stop()
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
