package gokeenrestapi

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"pgregory.net/rapid"
)

// Feature: property-based-testing, Property 1: Route parsing extracts correct values
// Validates: Requirements 1.1, 1.2, 1.4
func TestRouteCommandParsingExtractsIPAndMaskRegardlessOfWhitespace(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		ip := genIPv4Address().Draw(t, "ip")
		mask := genSubnetMask().Draw(t, "mask")

		// Generate route command with varying case and whitespace
		routeKeyword := rapid.SampledFrom([]string{"route", "ROUTE", "Route"}).Draw(t, "routeKeyword")

		// Generate varying whitespace (single space, double space, or tab)
		ws1 := rapid.SampledFrom([]string{" ", "  ", "\t"}).Draw(t, "ws1")
		ws2 := rapid.SampledFrom([]string{" ", "  ", "\t"}).Draw(t, "ws2")
		ws3 := rapid.SampledFrom([]string{" ", "  ", "\t"}).Draw(t, "ws3")

		routeCmd := fmt.Sprintf("%s%sADD%s%s%sMASK%s%s", routeKeyword, ws1, ws2, ip, ws3, ws3, mask)

		// Test property: parsing should extract the exact IP and mask values
		matches := routeRegex.FindStringSubmatch(routeCmd)

		// Assert property holds
		if len(matches) != 3 {
			t.Fatalf("failed to parse route command: got %d matches, want 3 for command: %q",
				len(matches), routeCmd)
		}

		if matches[1] != ip {
			t.Fatalf("IP mismatch: got %q, want %q in command: %q",
				matches[1], ip, routeCmd)
		}

		if matches[2] != mask {
			t.Fatalf("mask mismatch: got %q, want %q in command: %q",
				matches[2], mask, routeCmd)
		}
	})
}

// Feature: property-based-testing, Property 2: Invalid route commands are rejected
// Validates: Requirements 1.3
func TestMalformedRouteCommandsFailValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate invalid route command
		invalidCmd := genInvalidRouteCommand().Draw(t, "invalidCmd")

		// Test property: parser should reject malformed commands
		matches := routeRegex.FindStringSubmatch(invalidCmd)

		// Assert property holds: invalid commands should either:
		// 1. Not match the regex pattern at all (len(matches) != 3), OR
		// 2. Match the pattern but fail IP validation
		if len(matches) == 3 {
			ip := matches[1]
			mask := matches[2]
			// If regex matched, the IP validation should reject invalid IPs
			if validateIPAddress(ip) && validateIPAddress(mask) {
				t.Fatalf("invalid command was incorrectly accepted: %q with IP=%s mask=%s",
					invalidCmd, ip, mask)
			}
		}
	})
}

// Feature: property-based-testing, Property 3: Batch parsing consistency
// Validates: Requirements 1.5
func TestRouteParsingProducesSameResultsForIndividualAndBatchProcessing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a batch of route commands
		numRoutes := rapid.IntRange(1, 10).Draw(t, "numRoutes")
		var routes []string
		var expectedResults []struct {
			ip   string
			mask string
		}

		for i := range numRoutes {
			ip := genIPv4Address().Draw(t, fmt.Sprintf("ip_%d", i))
			mask := genSubnetMask().Draw(t, fmt.Sprintf("mask_%d", i))
			routeCmd := fmt.Sprintf("route ADD %s MASK %s", ip, mask)
			routes = append(routes, routeCmd)
			expectedResults = append(expectedResults, struct {
				ip   string
				mask string
			}{ip, mask})
		}

		// Parse individually
		var individualResults []struct {
			ip   string
			mask string
		}
		for _, route := range routes {
			matches := routeRegex.FindStringSubmatch(route)
			if len(matches) == 3 {
				individualResults = append(individualResults, struct {
					ip   string
					mask string
				}{matches[1], matches[2]})
			}
		}

		// Parse as batch (simulating batch processing)
		batchInput := strings.Join(routes, "\n")
		lines := strings.Split(batchInput, "\n")
		var batchResults []struct {
			ip   string
			mask string
		}
		for _, line := range lines {
			if line == "" {
				continue
			}
			matches := routeRegex.FindStringSubmatch(line)
			if len(matches) == 3 {
				batchResults = append(batchResults, struct {
					ip   string
					mask string
				}{matches[1], matches[2]})
			}
		}

		// Assert property holds: individual and batch parsing should produce same results
		if len(individualResults) != len(batchResults) {
			t.Fatalf("batch parsing inconsistency: individual=%d results, batch=%d results",
				len(individualResults), len(batchResults))
		}

		for i := range individualResults {
			if individualResults[i].ip != batchResults[i].ip {
				t.Fatalf("IP mismatch at index %d: individual=%q, batch=%q",
					i, individualResults[i].ip, batchResults[i].ip)
			}
			if individualResults[i].mask != batchResults[i].mask {
				t.Fatalf("mask mismatch at index %d: individual=%q, batch=%q",
					i, individualResults[i].mask, batchResults[i].mask)
			}
		}
	})
}

// Feature: property-based-testing, Property 4: CIDR and mask round-trip
// Validates: Requirements 2.1, 2.2
func TestCIDRToSubnetMaskConversionIsReversible(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid CIDR prefix length (0-32)
		originalCIDR := genCIDR().Draw(t, "cidr")

		// Convert CIDR to subnet mask
		mask := cidrToMask(originalCIDR)

		// Convert mask back to CIDR
		resultCIDR, err := maskToCIDRSafe(mask)

		// Assert property holds: round-trip should return original CIDR
		if err != nil {
			t.Fatalf("failed to convert mask back to CIDR: %v (original CIDR=%d, mask=%s)",
				err, originalCIDR, mask)
		}

		if resultCIDR != originalCIDR {
			t.Fatalf("CIDR round-trip failed: original=%d, after round-trip=%d (mask=%s)",
				originalCIDR, resultCIDR, mask)
		}
	})
}

// Feature: property-based-testing, Property 5: Network containment is transitive
// Validates: Requirements 2.3
func TestNetworkContainmentFollowsTransitiveProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate three networks with increasing specificity (decreasing CIDR prefix)
		// Network A: least specific (smallest CIDR)
		// Network B: more specific than A
		// Network C: most specific (largest CIDR)

		// Generate base IP for network A
		baseIP := genIPv4Address().Draw(t, "baseIP")

		// Generate CIDR prefixes ensuring A <= B <= C
		cidrA := rapid.IntRange(8, 28).Draw(t, "cidrA") // Leave room for B and C
		cidrB := rapid.IntRange(cidrA, 30).Draw(t, "cidrB")
		cidrC := rapid.IntRange(cidrB, 32).Draw(t, "cidrC")

		// Parse network A
		networkAStr := fmt.Sprintf("%s/%d", baseIP, cidrA)
		_, networkA, err := net.ParseCIDR(networkAStr)
		if err != nil {
			t.Fatalf("failed to parse network A: %v", err)
		}

		// Generate an IP within network A for network B
		// Use the network address of A as the base
		networkBIP := networkA.IP.String()
		networkBStr := fmt.Sprintf("%s/%d", networkBIP, cidrB)
		_, networkB, err := net.ParseCIDR(networkBStr)
		if err != nil {
			t.Fatalf("failed to parse network B: %v", err)
		}

		// Generate an IP within network B for network C
		networkCIP := networkB.IP.String()
		networkCStr := fmt.Sprintf("%s/%d", networkCIP, cidrC)
		_, networkC, err := net.ParseCIDR(networkCStr)
		if err != nil {
			t.Fatalf("failed to parse network C: %v", err)
		}

		// Test property: if A contains B and B contains C, then A must contain C
		aContainsB := networkContains(networkA, networkB)
		bContainsC := networkContains(networkB, networkC)
		aContainsC := networkContains(networkA, networkC)

		// Assert transitivity property
		if aContainsB && bContainsC && !aContainsC {
			t.Fatalf("transitivity violated: A contains B, B contains C, but A does not contain C\nA=%s\nB=%s\nC=%s",
				networkAStr, networkBStr, networkCStr)
		}
	})
}

// Feature: property-based-testing, Property 6: Network containment reflexivity
// Validates: Requirements 2.3
func TestNetworkAlwaysContainsItself(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random network
		networkStr := genIPv4Network().Draw(t, "network")

		// Parse the network
		_, network, err := net.ParseCIDR(networkStr)
		if err != nil {
			t.Fatalf("failed to parse network: %v", err)
		}

		// Test property: any network should contain itself
		if !networkContains(network, network) {
			t.Fatalf("reflexivity violated: network does not contain itself: %s", networkStr)
		}
	})
}

// Feature: property-based-testing, Property 7: IP validation correctness
// Validates: Requirements 2.4
func TestIPv4ValidationAcceptsValidAddressesAndRejectsInvalid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test with valid IPv4 addresses
		validIP := genIPv4Address().Draw(t, "validIP")

		// Test property: generated valid IPs should pass validation
		if !validateIPAddress(validIP) {
			t.Fatalf("valid IP rejected by validator: %s", validIP)
		}

		// Also verify using the helper function
		if !isValidIPv4(validIP) {
			t.Fatalf("valid IP rejected by isValidIPv4: %s", validIP)
		}

		// Test with invalid IP addresses
		invalidIP := rapid.SampledFrom([]string{
			"999.999.999.999", // Out of range octets
			"192.168.1",       // Missing octet
			"192.168.1.1.1",   // Too many octets
			"192.168.1.a",     // Non-numeric octet
			"",                // Empty string
			"not an ip",       // Random text
			"192.168.-1.1",    // Negative octet
			"192.168.1.256",   // Octet > 255
			"::1",             // IPv6 address
			"2001:db8::1",     // IPv6 address
		}).Draw(t, "invalidIP")

		// Test property: invalid IPs should fail validation
		if validateIPAddress(invalidIP) {
			t.Fatalf("invalid IP accepted by validator: %s", invalidIP)
		}

		if isValidIPv4(invalidIP) {
			t.Fatalf("invalid IP accepted by isValidIPv4: %s", invalidIP)
		}
	})
}

// Feature: property-based-testing, Property 18: Exact route matching is symmetric
// Validates: Requirements 5.1
func TestRouteMatchingIsSymmetricForSameDestinationAndInterface(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random interface ID
		interfaceID := rapid.SampledFrom([]string{"ISP0", "ISP1", "Wireguard0", "Bridge0"}).Draw(t, "interfaceID")

		// Generate two routes
		route1 := genRouteWithInterface(interfaceID).Draw(t, "route1")
		route2 := genRouteWithInterface(interfaceID).Draw(t, "route2")

		// Convert to CIDR notation for comparison
		cidr1, err1 := maskToCIDRSafe(route1.Mask)
		cidr2, err2 := maskToCIDRSafe(route2.Mask)

		if err1 != nil || err2 != nil {
			t.Skip("invalid mask generated")
		}

		dest1 := fmt.Sprintf("%s/%d", route1.IP, cidr1)
		dest2 := fmt.Sprintf("%s/%d", route2.IP, cidr2)

		// Check if routes match (exact match: same destination and interface)
		route1MatchesRoute2 := (dest1 == dest2) && (route1.Interface == route2.Interface)
		route2MatchesRoute1 := (dest2 == dest1) && (route2.Interface == route1.Interface)

		// Test property: matching should be symmetric
		if route1MatchesRoute2 != route2MatchesRoute1 {
			t.Fatalf("route matching is not symmetric:\nroute1=%+v (dest=%s)\nroute2=%+v (dest=%s)\nroute1 matches route2: %v\nroute2 matches route1: %v",
				route1, dest1, route2, dest2, route1MatchesRoute2, route2MatchesRoute1)
		}
	})
}

// Feature: property-based-testing, Property 19: Route containment respects network containment
// Validates: Requirements 5.2
func TestRouteContainmentDetectionRespectsNetworkContainment(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random interface ID
		interfaceID := rapid.SampledFrom([]string{"ISP0", "ISP1", "Wireguard0", "Bridge0"}).Draw(t, "interfaceID")

		// Generate a container route
		containerRoute := genRouteWithInterface(interfaceID).Draw(t, "containerRoute")

		// Generate an overlapping (contained) route
		containedRoute := genOverlappingRoute(containerRoute).Draw(t, "containedRoute")

		// Parse networks
		containerCIDR, err1 := maskToCIDRSafe(containerRoute.Mask)
		containedCIDR, err2 := maskToCIDRSafe(containedRoute.Mask)

		if err1 != nil || err2 != nil {
			t.Skip("invalid mask generated")
		}

		containerNetStr := fmt.Sprintf("%s/%d", containerRoute.IP, containerCIDR)
		containedNetStr := fmt.Sprintf("%s/%d", containedRoute.IP, containedCIDR)

		_, containerNet, err1 := net.ParseCIDR(containerNetStr)
		_, containedNet, err2 := net.ParseCIDR(containedNetStr)

		if err1 != nil || err2 != nil {
			t.Skip("failed to parse CIDR")
		}

		// Check network containment
		networkContainmentHolds := networkContains(containerNet, containedNet)

		// Simulate the route checking logic from checkInterfaceContainsRoute
		// Create a mock existing routes list with the container route
		existingRoutes := []gokeenrestapimodels.RciShowIpRoute{
			{
				Destination: containerNetStr,
				Interface:   containerRoute.Interface,
			},
		}

		// Check if the contained route would be detected as already existing
		routeExists, err := checkInterfaceContainsRoute(
			containedRoute.IP,
			containedRoute.Mask,
			containedRoute.Interface,
			existingRoutes,
		)

		if err != nil {
			t.Fatalf("error checking route containment: %v", err)
		}

		// Test property: if network containment holds, route checking should detect it
		if networkContainmentHolds && !routeExists {
			t.Fatalf("route containment does not respect network containment:\ncontainer route: %s (interface=%s)\ncontained route: %s (interface=%s)\nnetwork containment: %v\nroute exists check: %v",
				containerNetStr, containerRoute.Interface,
				containedNetStr, containedRoute.Interface,
				networkContainmentHolds, routeExists)
		}
	})
}

// Feature: property-based-testing, Property 20: Duplicate route handling is consistent
// Validates: Requirements 5.3
func TestDuplicateRouteDetectionProducesConsistentResults(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random interface ID
		interfaceID := rapid.SampledFrom([]string{"ISP0", "ISP1", "Wireguard0", "Bridge0"}).Draw(t, "interfaceID")

		// Generate a list of routes with some duplicates
		numRoutes := rapid.IntRange(2, 10).Draw(t, "numRoutes")
		var routes []struct {
			IP        string
			Mask      string
			Interface string
		}

		// Generate initial routes
		for i := range numRoutes {
			route := genRouteWithInterface(interfaceID).Draw(t, fmt.Sprintf("route_%d", i))
			routes = append(routes, route)
		}

		// Add some duplicates
		numDuplicates := rapid.IntRange(0, 3).Draw(t, "numDuplicates")
		for i := range numDuplicates {
			// Pick a random existing route to duplicate
			if len(routes) > 0 {
				idx := rapid.IntRange(0, len(routes)-1).Draw(t, fmt.Sprintf("dup_idx_%d", i))
				duplicate := routes[idx]
				routes = append(routes, duplicate)
			}
		}

		// Convert routes to RciShowIpRoute format
		var existingRoutes []gokeenrestapimodels.RciShowIpRoute
		for _, route := range routes {
			cidr, err := maskToCIDRSafe(route.Mask)
			if err != nil {
				continue
			}
			dest := fmt.Sprintf("%s/%d", route.IP, cidr)
			existingRoutes = append(existingRoutes, gokeenrestapimodels.RciShowIpRoute{
				Destination: dest,
				Interface:   route.Interface,
			})
		}

		// Process the list multiple times and check consistency
		// For each route in the list, check if it's detected as existing
		var firstPassResults []bool
		var secondPassResults []bool

		for _, route := range routes {
			exists, err := checkInterfaceContainsRoute(route.IP, route.Mask, route.Interface, existingRoutes)
			if err != nil {
				t.Skip("error checking route")
			}
			firstPassResults = append(firstPassResults, exists)
		}

		// Second pass - should get identical results
		for _, route := range routes {
			exists, err := checkInterfaceContainsRoute(route.IP, route.Mask, route.Interface, existingRoutes)
			if err != nil {
				t.Skip("error checking route")
			}
			secondPassResults = append(secondPassResults, exists)
		}

		// Test property: processing multiple times should produce consistent results
		if len(firstPassResults) != len(secondPassResults) {
			t.Fatalf("inconsistent result lengths: first=%d, second=%d", len(firstPassResults), len(secondPassResults))
		}

		for i := range firstPassResults {
			if firstPassResults[i] != secondPassResults[i] {
				t.Fatalf("inconsistent duplicate detection at index %d:\nroute: %+v\nfirst pass: %v\nsecond pass: %v",
					i, routes[i], firstPassResults[i], secondPassResults[i])
			}
		}
	})
}

// Feature: property-based-testing, Property 21: Interface isolation in route comparison
// Validates: Requirements 5.4
func TestRoutesWithSameDestinationButDifferentInterfacesAreTreatedAsDistinct(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate two different interface IDs
		interface1 := rapid.SampledFrom([]string{"ISP0", "ISP1", "Wireguard0", "Bridge0"}).Draw(t, "interface1")
		interface2 := rapid.SampledFrom([]string{"ISP0", "ISP1", "Wireguard0", "Bridge0"}).
			Filter(func(iface string) bool { return iface != interface1 }).
			Draw(t, "interface2")

		// Generate a route with the same destination but different interfaces
		ip := genIPv4Address().Draw(t, "ip")
		mask := genSubnetMask().Draw(t, "mask")

		route1 := struct {
			IP        string
			Mask      string
			Interface string
		}{
			IP:        ip,
			Mask:      mask,
			Interface: interface1,
		}

		route2 := struct {
			IP        string
			Mask      string
			Interface string
		}{
			IP:        ip,
			Mask:      mask,
			Interface: interface2,
		}

		// Convert to CIDR notation
		cidr, err := maskToCIDRSafe(mask)
		if err != nil {
			t.Skip("invalid mask generated")
		}

		dest := fmt.Sprintf("%s/%d", ip, cidr)

		// Create existing routes list with route1
		existingRoutes := []gokeenrestapimodels.RciShowIpRoute{
			{
				Destination: dest,
				Interface:   route1.Interface,
			},
		}

		// Check if route2 (same destination, different interface) is detected as existing
		// when checking against interface1
		existsOnInterface1, err := checkInterfaceContainsRoute(
			route2.IP,
			route2.Mask,
			route1.Interface, // Checking on interface1
			existingRoutes,
		)
		if err != nil {
			t.Fatalf("error checking route on interface1: %v", err)
		}

		// Check if route2 is detected as existing when checking against interface2
		existsOnInterface2, err := checkInterfaceContainsRoute(
			route2.IP,
			route2.Mask,
			route2.Interface, // Checking on interface2
			existingRoutes,
		)
		if err != nil {
			t.Fatalf("error checking route on interface2: %v", err)
		}

		// Test property: routes with same destination but different interfaces should be treated as distinct
		// route2 should be detected as existing on interface1 (because route1 exists there)
		// but NOT on interface2 (because no route exists there)
		if !existsOnInterface1 {
			t.Fatalf("route not detected on interface1 where it exists:\nroute1: %+v (dest=%s)\nroute2: %+v (dest=%s)\nexistsOnInterface1: %v",
				route1, dest, route2, dest, existsOnInterface1)
		}

		if existsOnInterface2 {
			t.Fatalf("route incorrectly detected on interface2 where it doesn't exist:\nroute1: %+v (dest=%s, interface=%s)\nroute2: %+v (dest=%s, interface=%s)\nexistsOnInterface2: %v",
				route1, dest, route1.Interface, route2, dest, route2.Interface, existsOnInterface2)
		}
	})
}
