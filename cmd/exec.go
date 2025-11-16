package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/spf13/cobra"
)

// execCommand executes a command on the Keenetic (Netcraze) router and returns the results
func execCommand(args []string) ([]gokeenrestapimodels.ParseResponse, error) {
	cmdToExecute := strings.Join(args, " ")
	parseC := gokeenrestapimodels.ParseRequest{Parse: cmdToExecute}
	return gokeenrestapi.Common.ExecutePostParse(parseC)
}

// printExecResults prints the execution results to stdout
func printExecResults(results []gokeenrestapimodels.ParseResponse) {
	gokeenlog.Info("Result:")
	for _, r := range results {
		if r.Parse.DynamicData != "" {
			fmt.Println(r.Parse.DynamicData)
		}
	}
}

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdExec,
		Aliases: AliasesExec,
		Short:   "Run custom router commands directly",
		Long: `Execute custom Keenetic (Netcraze) CLI commands directly on your router via REST API.

This command provides direct access to the Keenetic (Netcraze) router's command-line interface,
allowing you to run any supported CLI command. It's useful for advanced configuration
tasks not covered by other gokeenapi commands.

The command accepts Keenetic (Netcraze) CLI syntax and returns the router's response.
Multiple words are automatically joined with spaces to form the complete command.

Examples:
  # Show system information
  gokeenapi exec --config config.yaml show version

  # Display interface statistics
  gokeenapi exec --config config.yaml show interface

  # Show routing table
  gokeenapi exec --config config.yaml show ip route

  # Display WireGuard status
  gokeenapi exec --config config.yaml show interface Wireguard0

  # Execute configuration commands
  gokeenapi exec --config config.yaml interface Wireguard0 description "My VPN"

Warning: This command provides direct router access. Incorrect commands may
affect router functionality. Use with caution and ensure you understand
the Keenetic (Netcraze) CLI syntax before executing commands.`,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cmdToExecute := strings.Join(args, " ")
		return gokeenspinner.WrapWithSpinner(fmt.Sprintf("Executing %v command", color.CyanString(cmdToExecute)), func() error {
			result, err := execCommand(args)
			if err != nil {
				return err
			}
			printExecResults(result)
			return nil
		})
	}
	return cmd
}
