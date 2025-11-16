package cmd

import (
	"fmt"
	"regexp"

	"github.com/fatih/color"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"github.com/spf13/cobra"
)

func newDeleteKnownHostsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CmdDeleteKnownHosts,
		Aliases: AliasesDeleteKnownHosts,
		Short:   "Clean up device list using name or MAC patterns",
		Long: `Delete known hosts from your Keenetic (Netcraze) router using regex pattern matching.

This command removes devices from the router's known hosts list (hotspot database)
based on either hostname or MAC address patterns. Use regex patterns to match
multiple hosts at once or target specific devices.

The command will:
1. Retrieve all known hosts from the router
2. Apply the regex pattern to match hosts
3. Display matching hosts for review
4. Ask for confirmation (unless --force is used)
5. Delete the confirmed hosts

Examples:
  # Delete hosts with names containing "guest"
  gokeenapi delete-known-hosts --config config.yaml --name-pattern ".*guest.*"

  # Delete hosts with specific MAC prefix
  gokeenapi delete-known-hosts --config config.yaml --mac-pattern "^aa:bb:cc:.*"

  # Delete without confirmation
  gokeenapi delete-known-hosts --config config.yaml --name-pattern "temp.*" --force

  # Delete hosts with exact name match
  gokeenapi delete-known-hosts --config config.yaml --name-pattern "^old-device$"

Note: Exactly one of --name-pattern or --mac-pattern must be specified.`,
	}

	var namePattern, macPattern string
	var force bool
	cmd.Flags().StringVar(&namePattern, "name-pattern", "",
		`Regex pattern to match against host names for deletion.
Examples: ".*guest.*" (contains guest), "^temp" (starts with temp)
Cannot be used together with --mac-pattern.`)
	cmd.Flags().StringVar(&macPattern, "mac-pattern", "",
		`Regex pattern to match against host MAC addresses for deletion.
Examples: "^aa:bb:cc:" (MAC prefix), ".*:.*:.*:dd:ee:ff$" (MAC suffix)
Cannot be used together with --name-pattern.`)
	cmd.Flags().BoolVar(&force, "force", false,
		`Skip confirmation prompt and delete hosts immediately.
Use with caution as this bypasses the safety confirmation.`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if (namePattern == "") == (macPattern == "") {
			return fmt.Errorf("exactly one of --name-pattern or --mac-pattern must be specified")
		}

		var regex *regexp.Regexp
		var err error
		if namePattern != "" {
			regex, err = regexp.Compile(namePattern)
		} else {
			regex, err = regexp.Compile(macPattern)
		}
		if err != nil {
			return err
		}

		hotspot, err := gokeenrestapi.Ip.GetAllHotspots()
		if err != nil {
			return err
		}

		var hostMacsToDelete []string
		for _, host := range hotspot.Host {
			var matched bool
			if namePattern != "" {
				matched = regex.MatchString(host.Name)
			} else {
				matched = regex.MatchString(host.Mac)
			}
			if matched {
				gokeenlog.InfoSubStepf("Matching host: %v (MAC: %v)", color.CyanString(host.Name), color.BlueString(host.Mac))
				hostMacsToDelete = append(hostMacsToDelete, host.Mac)
			}
		}

		if len(hostMacsToDelete) == 0 {
			gokeenlog.Info("No hosts found matching the pattern, no need to delete")
			return nil
		}

		if !force {
			confirmed, err := confirmAction(fmt.Sprintf("\nFound %d host(s) to delete. Do you want to continue?", len(hostMacsToDelete)))
			if err != nil {
				return err
			}
			if !confirmed {
				gokeenlog.Info("Deletion cancelled")
				return nil
			}
		}

		return gokeenrestapi.Ip.DeleteKnownHosts(hostMacsToDelete)
	}
	return cmd
}
