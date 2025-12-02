package cmd

import (
	"fmt"
	"slices"

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
		Short:   "Remove DNS-routing rules from specified interfaces",
		Long: `Delete DNS-routing rules (domain-based routing policies) from your Keenetic router.

This command removes DNS-routing rules from specified interfaces. By default, it processes
all interfaces defined in your configuration file's 'dns.routes.groups' section. You can
target a specific interface using the --interface-id flag.

The command will:
1. Check router firmware version (requires 5.0.1+)
2. Fetch DNS-routing groups from the router for target interfaces
3. List all rules to be deleted (dns-proxy routes and object-groups)
4. Ask for confirmation (unless --force is used)
5. Remove dns-proxy routes first, then object-groups
6. Save router configuration

Examples:
  # Delete DNS-routing rules from all interfaces in config
  gokeenapi delete-dns-routing --config config.yaml

  # Delete DNS-routing rules for specific interface
  gokeenapi delete-dns-routing --config config.yaml --interface-id Wireguard0

  # Delete without confirmation prompt
  gokeenapi delete-dns-routing --config config.yaml --force

Safety: Similar to 'delete-routes', this removes DNS-routing configuration for
specified interfaces. Only groups routed through target interfaces are deleted.

Requirements: Keenetic firmware version 5.0.1 or higher`,
	}

	var interfaceId string
	var force bool
	cmd.Flags().StringVar(&interfaceId, "interface-id", "",
		`Target a specific Keenetic interface ID for DNS-routing deletion.
If not specified, processes all interfaces from the config file.
Use 'show-interfaces' to list available interface IDs.`)
	cmd.Flags().BoolVar(&force, "force", false,
		`Skip confirmation prompt and delete DNS-routing rules immediately.
Use with caution as this bypasses the safety confirmation.`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Check router version support
		if err := gokeenrestapi.DnsRouting.CheckDnsRoutingSupport(); err != nil {
			return err
		}

		// Determine which interfaces to process
		var targetInterfaces []string
		if interfaceId != "" {
			// Specific interface requested
			targetInterfaces = append(targetInterfaces, interfaceId)
		} else {
			// Use interfaces from config file
			for _, group := range config.Cfg.DNS.Routes.Groups {
				// Collect unique interface IDs
				found := slices.Contains(targetInterfaces, group.InterfaceID)
				if !found {
					targetInterfaces = append(targetInterfaces, group.InterfaceID)
				}
			}
		}

		if len(targetInterfaces) == 0 {
			gokeenlog.Info("No DNS-routing interfaces defined in configuration")
			return nil
		}

		// Validate all target interfaces
		for _, ifaceId := range targetInterfaces {
			if err := gokeenrestapi.Checks.CheckInterfaceId(ifaceId); err != nil {
				return err
			}
			if err := gokeenrestapi.Checks.CheckInterfaceExists(ifaceId); err != nil {
				return err
			}
		}

		// Get all existing DNS-routing groups from router
		existingGroups, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		if err != nil {
			return err
		}

		if len(existingGroups) == 0 {
			gokeenlog.Info("No DNS-routing groups found on router")
			return nil
		}

		// Get existing dns-proxy routes to filter by interface
		existingRoutes, err := gokeenrestapi.DnsRouting.GetExistingDnsProxyRoutes()
		if err != nil {
			return err
		}

		// Convert existing groups to config format for deletion
		// Only include groups that are routed through target interfaces
		var groupsToDelete []config.DnsRoutingGroup
		totalDomains := 0
		for groupName, domains := range existingGroups {
			routeInterface, exists := existingRoutes[groupName]
			if !exists {
				continue
			}

			// Check if this group's interface is in our target list
			isTargetInterface := slices.Contains(targetInterfaces, routeInterface)

			if !isTargetInterface {
				continue
			}

			totalDomains += len(domains)
			groupsToDelete = append(groupsToDelete, config.DnsRoutingGroup{
				Name:        groupName,
				InterfaceID: routeInterface,
			})

			// Display group info
			gokeenlog.InfoSubStepf("DNS-routing group to delete: %v (interface: %v, domains: %v)",
				color.CyanString(groupName),
				color.YellowString(routeInterface),
				color.BlueString("%d", len(domains)))

			// Show domains in the group
			for _, domain := range domains {
				gokeenlog.InfoSubStepf("  - %v", color.MagentaString(domain))
			}
		}

		if len(groupsToDelete) == 0 {
			if interfaceId != "" {
				gokeenlog.Infof("No DNS-routing groups found for interface %v", color.YellowString(interfaceId))
			} else {
				gokeenlog.Info("No DNS-routing groups found for configured interfaces")
			}
			return nil
		}

		// Request confirmation unless --force is used
		confirmMsg := fmt.Sprintf("\nFound %v DNS-routing group(s) with %v total domain(s) to delete. Do you want to continue?",
			color.CyanString("%v", len(groupsToDelete)),
			color.CyanString("%v", totalDomains))

		if !force {
			confirmed, err := confirmAction(confirmMsg)
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
