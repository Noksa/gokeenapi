package gokeenrestapi

import (
	"fmt"
	"net"

	"pgregory.net/rapid"
)

// Generator utilities for property-based testing

// genIPv4Address generates valid IPv4 addresses in dotted-decimal notation
func genIPv4Address() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		octet1 := rapid.IntRange(0, 255).Draw(t, "octet1")
		octet2 := rapid.IntRange(0, 255).Draw(t, "octet2")
		octet3 := rapid.IntRange(0, 255).Draw(t, "octet3")
		octet4 := rapid.IntRange(0, 255).Draw(t, "octet4")
		return fmt.Sprintf("%d.%d.%d.%d", octet1, octet2, octet3, octet4)
	})
}

// genSubnetMask generates valid subnet masks in dotted-decimal notation
func genSubnetMask() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate CIDR prefix length (0-32) and convert to mask
		cidr := rapid.IntRange(0, 32).Draw(t, "cidr")
		return cidrToMask(cidr)
	})
}

// genCIDR generates valid CIDR prefix lengths (0-32)
func genCIDR() *rapid.Generator[int] {
	return rapid.IntRange(0, 32)
}

// genIPv4Network generates valid IPv4 networks in CIDR notation
func genIPv4Network() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		ip := genIPv4Address().Draw(t, "ip")
		cidr := genCIDR().Draw(t, "cidr")
		return fmt.Sprintf("%s/%d", ip, cidr)
	})
}

// genRouteCommand generates valid Windows route commands
func genRouteCommand() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		ip := genIPv4Address().Draw(t, "ip")
		mask := genSubnetMask().Draw(t, "mask")
		// Generate with varying case for "route"
		routeKeyword := rapid.SampledFrom([]string{"route", "ROUTE", "Route"}).Draw(t, "routeKeyword")
		return fmt.Sprintf("%s ADD %s MASK %s", routeKeyword, ip, mask)
	})
}

// genRouteCommandWithWhitespace generates route commands with varying whitespace
func genRouteCommandWithWhitespace() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		ip := genIPv4Address().Draw(t, "ip")
		mask := genSubnetMask().Draw(t, "mask")

		// Generate varying amounts of whitespace
		ws1 := rapid.SampledFrom([]string{" ", "  ", "\t", "   "}).Draw(t, "ws1")
		ws2 := rapid.SampledFrom([]string{" ", "  ", "\t", "   "}).Draw(t, "ws2")
		ws3 := rapid.SampledFrom([]string{" ", "  ", "\t", "   "}).Draw(t, "ws3")

		return fmt.Sprintf("route%sADD%s%s%sMASK%s%s", ws1, ws2, ip, ws3, ws3, mask)
	})
}

// genInvalidRouteCommand generates malformed route commands for negative testing
func genInvalidRouteCommand() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"route ADD",             // Missing IP and mask
		"route ADD 192.168.1.1", // Missing mask
		"route ADD 999.999.999.999 MASK 255.255.255.0", // Invalid IP
		"route ADD 192.168.1.1 MASK 999.999.999.999",   // Invalid mask
		"ADD 192.168.1.1 MASK 255.255.255.0",           // Missing route keyword
		"192.168.1.1 MASK 255.255.255.0",               // Missing route and ADD
		"route 192.168.1.1 255.255.255.0",              // Missing ADD and MASK
		"",                                             // Empty string
		"random text",                                  // Completely invalid
	})
}

// Helper functions for validation and comparison

// isValidIPv4 checks if a string is a valid IPv4 address
func isValidIPv4(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	// Ensure it's IPv4 (not IPv6)
	return parsed.To4() != nil
}

// isValidSubnetMask checks if a string is a valid subnet mask
func isValidSubnetMask(mask string) bool {
	ip := net.ParseIP(mask)
	if ip == nil {
		return false
	}

	ipv4Mask := net.IPMask(ip.To4())
	if ipv4Mask == nil {
		return false
	}

	// Check if it's a valid contiguous mask
	ones, bits := ipv4Mask.Size()
	return bits == 32 && ones >= 0 && ones <= 32
}

// isValidCIDR checks if an integer is a valid CIDR prefix length
func isValidCIDR(cidr int) bool {
	return cidr >= 0 && cidr <= 32
}

// cidrToMask converts a CIDR prefix length to a dotted-decimal subnet mask
func cidrToMask(cidr int) string {
	if cidr < 0 || cidr > 32 {
		return ""
	}

	mask := net.CIDRMask(cidr, 32)
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

// maskToCIDRSafe safely converts a subnet mask to CIDR notation
func maskToCIDRSafe(mask string) (int, error) {
	ip := net.ParseIP(mask)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP: %s", mask)
	}

	ipv4Mask := net.IPMask(ip.To4())
	if ipv4Mask == nil {
		return 0, fmt.Errorf("not an IPv4 mask: %s", mask)
	}

	ones, _ := ipv4Mask.Size()
	return ones, nil
}

// networksEqual checks if two networks are equal
func networksEqual(net1, net2 *net.IPNet) bool {
	if net1 == nil || net2 == nil {
		return net1 == net2
	}
	return net1.IP.Equal(net2.IP) && net1.Mask.String() == net2.Mask.String()
}

// networkContains checks if network1 contains network2
func networkContains(network1, network2 *net.IPNet) bool {
	if network1 == nil || network2 == nil {
		return false
	}

	// Check if network1 contains the first IP of network2
	if !network1.Contains(network2.IP) {
		return false
	}

	// Check if network1's mask is less specific (smaller prefix) than network2's
	ones1, _ := network1.Mask.Size()
	ones2, _ := network2.Mask.Size()

	return ones1 <= ones2
}

// parseNetwork parses a CIDR notation string into a *net.IPNet
func parseNetwork(cidr string) (*net.IPNet, error) {
	_, network, err := net.ParseCIDR(cidr)
	return network, err
}

// Route deduplication test generators

// genRouteWithInterface generates a route (IP + mask) with a configurable interface
func genRouteWithInterface(interfaceID string) *rapid.Generator[struct {
	IP        string
	Mask      string
	Interface string
}] {
	return rapid.Custom(func(t *rapid.T) struct {
		IP        string
		Mask      string
		Interface string
	} {
		ip := genIPv4Address().Draw(t, "ip")
		mask := genSubnetMask().Draw(t, "mask")
		return struct {
			IP        string
			Mask      string
			Interface string
		}{
			IP:        ip,
			Mask:      mask,
			Interface: interfaceID,
		}
	})
}

// genOverlappingRoute generates a route that overlaps with (is contained by) a given route
func genOverlappingRoute(container struct {
	IP        string
	Mask      string
	Interface string
}) *rapid.Generator[struct {
	IP        string
	Mask      string
	Interface string
}] {
	return rapid.Custom(func(t *rapid.T) struct {
		IP        string
		Mask      string
		Interface string
	} {
		// Parse the container network
		containerCIDR, err := maskToCIDRSafe(container.Mask)
		if err != nil {
			// Generate a random route instead
			ip := genIPv4Address().Draw(t, "fallback_ip")
			mask := genSubnetMask().Draw(t, "fallback_mask")
			return struct {
				IP        string
				Mask      string
				Interface string
			}{
				IP:        ip,
				Mask:      mask,
				Interface: container.Interface,
			}
		}

		containerNetStr := fmt.Sprintf("%s/%d", container.IP, containerCIDR)
		_, containerNet, err := net.ParseCIDR(containerNetStr)
		if err != nil {
			// Generate a random route instead
			ip := genIPv4Address().Draw(t, "fallback_ip")
			mask := genSubnetMask().Draw(t, "fallback_mask")
			return struct {
				IP        string
				Mask      string
				Interface string
			}{
				IP:        ip,
				Mask:      mask,
				Interface: container.Interface,
			}
		}

		// Generate a more specific CIDR (larger prefix) within the container
		// Ensure we have room to be more specific
		if containerCIDR >= 32 {
			// Can't be more specific than /32, return the same route
			// But we need to draw something to satisfy rapid
			_ = rapid.Bool().Draw(t, "dummy")
			return container
		}

		// Generate a CIDR between containerCIDR+1 and 32
		newCIDR := rapid.IntRange(containerCIDR+1, 32).Draw(t, "newCIDR")

		// Use the container's network IP as the base for the overlapping route
		newIP := containerNet.IP.String()
		newMask := cidrToMask(newCIDR)

		return struct {
			IP        string
			Mask      string
			Interface string
		}{
			IP:        newIP,
			Mask:      newMask,
			Interface: container.Interface,
		}
	})
}
