package cmd

import (
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
)

func newAddDnsRoutingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdAddDnsRouting,
		Aliases: AliasesAddDnsRouting,
		Short:   "Add DNS-routing rules for domain-based routing policies",
		Long: `Add DNS-routing rules to your Keenetic (Netcraze) router for domain-based routing.

This command creates domain groups (object-groups) and associates them with routing 
policies (dns-proxy routes), allowing you to route traffic for specific domains through 
designated network interfaces.

DNS-routing enables fine-grained control over traffic routing based on DNS queries:
- Route specific domains through VPN interfaces
- Direct streaming services through specific connections
- Implement split-tunneling for domain-based policies
- Organize domains into logical groups

Requirements:
- Keenetic firmware version 5.0.1 or higher
- Valid interface IDs (use 'show-interfaces' to verify)

Examples:
  # Add all DNS-routing rules from config file
  gokeenapi add-dns-routing --config config.yaml

  # Example config entries:
  # dns:
  #   routes:
  #     groups:
  #       - name: social-media
  #         domains:
  #           - facebook.com
  #           - instagram.com
  #         interfaceId: Wireguard0

The command automatically validates interface IDs and saves the configuration 
after adding rules.`,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return gokeenrestapi.DnsRouting.AddDnsRoutingGroups(config.Cfg.DNS.Routes.Groups)
	}

	return cmd
}
