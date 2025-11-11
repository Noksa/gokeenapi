package cmd

import (
	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenversion"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   CmdVersion,
		Short: "Show version information",
		Long:  `Display the current version and build date of gokeenapi.`,
		Run: func(cmd *cobra.Command, args []string) {
			gokeenlog.Infof("🚀  %v: %v, %v: %v",
				color.BlueString("Version"),
				color.CyanString(gokeenversion.Version()),
				color.BlueString("Build date"),
				color.CyanString(gokeenversion.BuildDate()))
		},
	}
	return cmd
}
