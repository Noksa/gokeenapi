package keenspinner

import (
	"fmt"
	"github.com/briandowns/spinner"
	"time"
)

func WrapWithSpinner(spinnerText string, f func() error) error {
	s := spinner.New(spinner.CharSets[70], 100*time.Millisecond)
	startTime := time.Now()
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
