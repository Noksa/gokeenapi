package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
)

func newDeleteDnsRoutingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdDeleteDnsRouting,
		Aliases: AliasesDeleteDnsRouting,
		Short:   "Remove DNS-routing rules from your router",
		Long: `Delete DNS-routing rules (domain-based routing policies) from your Keenetic router.

This command removes DNS-routing rules that match the entries defined in your configuration
file's 'dns.routes.groups' section. Only rules that currently exist in the router 
configuration will be deleted.

The command will:
1. Check router firmware version (requires 5.0.1+)
2. Fetch current router configuration for matching DNS-routing groups
3. List all rules to be deleted (dns-proxy routes and object-groups)
4. Ask for confirmation (unless --force is used)
5. Remove dns-proxy routes first, then object-groups
6. Save router configuration

Examples:
  # Delete DNS-routing rules matching config file entries
  gokeenapi delete-dns-routing --config config.yaml

  # Delete without confirmation prompt
  gokeenapi delete-dns-routing --config config.yaml --force

Safety: Only DNS-routing rules that match your config file entries are deleted.
Other routing rules in the router remain untouched.

Requirements: Keenetic firmware version 5.0.1 or higher`,
	}

	var force bool
	cmd.Flags().BoolVar(&force, "force", false,
		`Skip confirmation prompt and delete DNS-routing rules immediately.
Use with caution as this bypasses the safety confirmation.`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Check router version support
		if err := gokeenrestapi.DnsRouting.CheckDnsRoutingSupport(); err != nil {
			return err
		}

		// Get groups from configuration
		groups := config.Cfg.DNS.Routes.Groups
		if len(groups) == 0 {
			gokeenlog.Info("No DNS-routing groups defined in configuration")
			return nil
		}

		// Get existing DNS-routing groups from router
		existingGroups, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		if err != nil {
			return err
		}

		// Find matching groups to delete
		var groupsToDelete []config.DnsRoutingGroup
		for _, group := range groups {
			if _, exists := existingGroups[group.Name]; exists {
				groupsToDelete = append(groupsToDelete, group)
			}
		}

		if len(groupsToDelete) == 0 {
			gokeenlog.Info("No matching DNS-routing groups found to delete")
			return nil
		}

		// Display rules to be deleted
		for _, group := range groupsToDelete {
			// Get domain count from existing groups
			domainCount := 0
			if domains, exists := existingGroups[group.Name]; exists {
				domainCount = len(domains)
			}

			gokeenlog.InfoSubStepf("DNS-routing group to delete: %v (interface: %v, domains: %v)",
				color.CyanString(group.Name),
				color.YellowString(group.InterfaceID),
				color.BlueString("%d", domainCount))

			// Show domains in the group
			if domains, exists := existingGroups[group.Name]; exists {
				for _, domain := range domains {
					gokeenlog.InfoSubStepf("  - %v", color.MagentaString(domain))
				}
			}
		}

		// Request confirmation unless --force is used
		if !force {
			confirmed, err := confirmAction(fmt.Sprintf("\nFound %v DNS-routing group(s) to delete. Do you want to continue?", len(groupsToDelete)))
			if err != nil {
				return err
			}
			if !confirmed {
				gokeenlog.Info("Deletion cancelled")
				return nil
			}
		}

		// Delete the groups
		return gokeenrestapi.DnsRouting.DeleteDnsRoutingGroups(groupsToDelete)
	}
	return cmd
}
