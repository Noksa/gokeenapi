package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/spf13/cobra"
)

func newAddDnsRecordsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdAddDnsRecords,
		Aliases: AliasesAddDnsRecords,
		Short:   "Create custom DNS entries for local domains",
		Long: `Add static DNS records to your Keenetic (Netcraze) router's local DNS resolver.

This command creates custom DNS entries that resolve domain names to specific IP addresses
within your local network. Records are defined in the 'dns.records' section of your 
configuration file.

Each DNS record can map a single domain to multiple IP addresses, useful for:
- Local service discovery
- Custom domain resolution
- Load balancing between multiple servers
- Overriding external DNS resolution

Examples:
  # Add all DNS records from config file
  gokeenapi add-dns-records --config config.yaml

  # Example config entries:
  # dns:
  #   records:
  #     - domain: myserver.local
  #       ip: [192.168.1.100, 192.168.1.101]

The command automatically saves the configuration after adding records.`,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		//runningConfig, err := gokeenrestapi.Common.ShowRunningConfig()
		//if err != nil {
		//	return err
		//}
		var parseC []gokeenrestapimodels.ParseRequest
		for _, addDnsRecordSetting := range config.Cfg.DNS.Records {
			for _, ip := range addDnsRecordSetting.IP {
				c := fmt.Sprintf("ip host %v %v", addDnsRecordSetting.Domain, ip)
				parseC = append(parseC, gokeenrestapimodels.ParseRequest{Parse: c})
			}
		}
		if len(parseC) == 0 {
			return nil
		}
		err := gokeenspinner.WrapWithSpinner(fmt.Sprintf("Adding %v DNS records", color.CyanString("%v", len(parseC))), func() error {
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
