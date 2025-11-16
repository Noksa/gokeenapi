package cmd

import (
	"fmt"
	"slices"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/spf13/cobra"
)

func newDeleteDnsRecordsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdDeleteDnsRecords,
		Aliases: AliasesDeleteDnsRecords,
		Short:   "Remove custom DNS entries from your router",
		Long: `Delete static DNS records from your Keenetic (Netcraze) router's local DNS resolver.

This command removes DNS records that match the entries defined in your configuration
file's 'dns.records' section. Only records that currently exist in the router 
configuration will be deleted.

The command will:
1. Check current router configuration for matching DNS records
2. List all records to be deleted
3. Ask for confirmation (unless --force is used)
4. Remove the confirmed DNS records

Examples:
  # Delete DNS records matching config file entries
  gokeenapi delete-dns-records --config config.yaml

  # Delete without confirmation prompt
  gokeenapi delete-dns-records --config config.yaml --force

Safety: Only DNS records that exactly match your config file entries are deleted.
Other DNS records in the router remain untouched.`,
	}

	var force bool
	cmd.Flags().BoolVar(&force, "force", false,
		`Skip confirmation prompt and delete DNS records immediately.
Use with caution as this bypasses the safety confirmation.`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		runningConfig, err := gokeenrestapi.Common.ShowRunningConfig()
		if err != nil {
			return err
		}
		var parseC []gokeenrestapimodels.ParseRequest
		for _, addDnsRecordSetting := range config.Cfg.DNS.Records {
			for _, ip := range addDnsRecordSetting.IP {
				c := fmt.Sprintf("ip host %v %v", addDnsRecordSetting.Domain, ip)
				if !slices.Contains(runningConfig.Message, c) {
					continue
				}
				gokeenlog.InfoSubStepf("DNS record to delete: %v -> %v",
					color.CyanString(addDnsRecordSetting.Domain),
					color.BlueString(ip))
				c = fmt.Sprintf("no %v", c)
				parseC = append(parseC, gokeenrestapimodels.ParseRequest{Parse: c})
			}
		}
		if len(parseC) == 0 {
			gokeenlog.Info("No DNS records found to delete")
			return nil
		}

		if !force {
			confirmed, err := confirmAction(fmt.Sprintf("\nFound %v DNS record(s) to delete. Do you want to continue?", len(parseC)))
			if err != nil {
				return err
			}
			if !confirmed {
				gokeenlog.Info("Deletion cancelled")
				return nil
			}
		}
		err = gokeenspinner.WrapWithSpinner(fmt.Sprintf("Deleting %v DNS records", color.CyanString("%v", len(parseC))), func() error {
			parseC = gokeenrestapi.Common.EnsureSaveConfigAtEnd(parseC)
			result, err := gokeenrestapi.Common.ExecutePostParse(parseC...)
			if err != nil {
				return err
			}
			gokeenlog.PrintParseResponse(result)
			return err
		})
		return err
	}
	return cmd
}
