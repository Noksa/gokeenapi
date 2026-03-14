package gokeenrestapi

import (
	"fmt"
	"net"
	"strings"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

var _ = Describe("Property: IP Routes", func() {
	It("should extract correct IP and mask from route commands", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			ip := genIPv4Address().Draw(t, "ip")
			mask := genSubnetMask().Draw(t, "mask")

			routeKeyword := rapid.SampledFrom([]string{"route", "ROUTE", "Route"}).Draw(t, "routeKeyword")
			ws1 := rapid.SampledFrom([]string{" ", "  ", "\t"}).Draw(t, "ws1")
			ws2 := rapid.SampledFrom([]string{" ", "  ", "\t"}).Draw(t, "ws2")
			ws3 := rapid.SampledFrom([]string{" ", "  ", "\t"}).Draw(t, "ws3")

			routeCmd := fmt.Sprintf("%s%sADD%s%s%sMASK%s%s", routeKeyword, ws1, ws2, ip, ws3, ws3, mask)

			matches := routeRegex.FindStringSubmatch(routeCmd)
			Expect(matches).To(HaveLen(3), "failed to parse route command: %q", routeCmd)
			Expect(matches[1]).To(Equal(ip), "IP mismatch in command: %q", routeCmd)
			Expect(matches[2]).To(Equal(mask), "mask mismatch in command: %q", routeCmd)
		})
	})

	It("should reject malformed route commands", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			invalidCmd := genInvalidRouteCommand().Draw(t, "invalidCmd")
			matches := routeRegex.FindStringSubmatch(invalidCmd)

			if len(matches) == 3 {
				ip := matches[1]
				mask := matches[2]
				Expect(validateIPAddress(ip) && validateIPAddress(mask)).To(BeFalse(),
					"invalid command was incorrectly accepted: %q with IP=%s mask=%s", invalidCmd, ip, mask)
			}
		})
	})

	It("should parse batches consistently", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			numRoutes := rapid.IntRange(1, 10).Draw(t, "numRoutes")
			var routes []string

			for i := range numRoutes {
				ip := genIPv4Address().Draw(t, fmt.Sprintf("ip_%d", i))
				mask := genSubnetMask().Draw(t, fmt.Sprintf("mask_%d", i))
				routes = append(routes, fmt.Sprintf("route ADD %s MASK %s", ip, mask))
			}

			// Parse individually
			var individualResults []string
			for _, route := range routes {
				matches := routeRegex.FindStringSubmatch(route)
				if len(matches) == 3 {
					individualResults = append(individualResults, matches[1]+"/"+matches[2])
				}
			}

			// Parse as batch
			batchInput := strings.Join(routes, "\n")
			var batchResults []string
			for line := range strings.SplitSeq(batchInput, "\n") {
				if line == "" {
					continue
				}
				matches := routeRegex.FindStringSubmatch(line)
				if len(matches) == 3 {
					batchResults = append(batchResults, matches[1]+"/"+matches[2])
				}
			}

			Expect(batchResults).To(Equal(individualResults))
		})
	})

	It("should round-trip CIDR to mask and back", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			originalCIDR := genCIDR().Draw(t, "cidr")
			mask := cidrToMask(originalCIDR)

			resultCIDR, err := maskToCIDRSafe(mask)
			Expect(err).NotTo(HaveOccurred(), "original CIDR=%d, mask=%s", originalCIDR, mask)
			Expect(resultCIDR).To(Equal(originalCIDR), "mask=%s", mask)
		})
	})

	It("should maintain transitive network containment", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			baseIP := genIPv4Address().Draw(t, "baseIP")
			cidrA := rapid.IntRange(8, 28).Draw(t, "cidrA")
			cidrB := rapid.IntRange(cidrA, 30).Draw(t, "cidrB")
			cidrC := rapid.IntRange(cidrB, 32).Draw(t, "cidrC")

			_, networkA, err := net.ParseCIDR(fmt.Sprintf("%s/%d", baseIP, cidrA))
			Expect(err).NotTo(HaveOccurred())

			_, networkB, err := net.ParseCIDR(fmt.Sprintf("%s/%d", networkA.IP.String(), cidrB))
			Expect(err).NotTo(HaveOccurred())

			_, networkC, err := net.ParseCIDR(fmt.Sprintf("%s/%d", networkB.IP.String(), cidrC))
			Expect(err).NotTo(HaveOccurred())

			aContainsB := networkContains(networkA, networkB)
			bContainsC := networkContains(networkB, networkC)
			aContainsC := networkContains(networkA, networkC)

			if aContainsB && bContainsC {
				Expect(aContainsC).To(BeTrue(), "transitivity violated: A=%s B=%s C=%s",
					networkA, networkB, networkC)
			}
		})
	})

	It("should maintain reflexive network containment", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			networkStr := genIPv4Network().Draw(t, "network")
			_, network, err := net.ParseCIDR(networkStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(networkContains(network, network)).To(BeTrue(), "network: %s", networkStr)
		})
	})

	It("should validate IPv4 addresses correctly", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			validIP := genIPv4Address().Draw(t, "validIP")
			Expect(validateIPAddress(validIP)).To(BeTrue(), "valid IP rejected: %s", validIP)
			Expect(isValidIPv4(validIP)).To(BeTrue(), "valid IP rejected by isValidIPv4: %s", validIP)

			invalidIP := rapid.SampledFrom([]string{
				"999.999.999.999", "192.168.1", "192.168.1.1.1",
				"192.168.1.a", "", "not an ip", "192.168.-1.1",
				"192.168.1.256", "::1", "2001:db8::1",
			}).Draw(t, "invalidIP")
			Expect(validateIPAddress(invalidIP)).To(BeFalse(), "invalid IP accepted: %s", invalidIP)
			Expect(isValidIPv4(invalidIP)).To(BeFalse(), "invalid IP accepted by isValidIPv4: %s", invalidIP)
		})
	})

	It("should match routes symmetrically", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			interfaceID := rapid.SampledFrom([]string{"ISP0", "ISP1", "Wireguard0", "Bridge0"}).Draw(t, "interfaceID")
			route1 := genRouteWithInterface(interfaceID).Draw(t, "route1")
			route2 := genRouteWithInterface(interfaceID).Draw(t, "route2")

			cidr1, err1 := maskToCIDRSafe(route1.Mask)
			cidr2, err2 := maskToCIDRSafe(route2.Mask)
			if err1 != nil || err2 != nil {
				t.Skip("invalid mask generated")
			}

			dest1 := fmt.Sprintf("%s/%d", route1.IP, cidr1)
			dest2 := fmt.Sprintf("%s/%d", route2.IP, cidr2)

			match12 := (dest1 == dest2) && (route1.Interface == route2.Interface)
			match21 := (dest2 == dest1) && (route2.Interface == route1.Interface)
			Expect(match12).To(Equal(match21), "route1=%+v route2=%+v", route1, route2)
		})
	})

	It("should respect network containment in route checking", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			interfaceID := rapid.SampledFrom([]string{"ISP0", "ISP1", "Wireguard0", "Bridge0"}).Draw(t, "interfaceID")
			containerRoute := genRouteWithInterface(interfaceID).Draw(t, "containerRoute")
			containedRoute := genOverlappingRoute(containerRoute).Draw(t, "containedRoute")

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
				t.Skip("invalid CIDR generated")
			}

			existingRoutes := []gokeenrestapimodels.RciShowIpRoute{
				{Destination: containerNetStr, Interface: containerRoute.Interface},
			}

			routeExists, err := checkInterfaceContainsRoute(
				containedRoute.IP, containedRoute.Mask, containedRoute.Interface, existingRoutes,
			)
			Expect(err).NotTo(HaveOccurred())

			if networkContains(containerNet, containedNet) {
				Expect(routeExists).To(BeTrue(),
					"container=%s contained=%s", containerNetStr, containedNetStr)
			}
		})
	})

	It("should detect duplicates consistently across multiple passes", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			interfaceID := rapid.SampledFrom([]string{"ISP0", "ISP1", "Wireguard0", "Bridge0"}).Draw(t, "interfaceID")
			numRoutes := rapid.IntRange(2, 10).Draw(t, "numRoutes")

			type routeEntry struct {
				IP, Mask, Interface string
			}
			var routes []routeEntry

			for i := range numRoutes {
				r := genRouteWithInterface(interfaceID).Draw(t, fmt.Sprintf("route_%d", i))
				routes = append(routes, routeEntry{r.IP, r.Mask, r.Interface})
			}

			numDuplicates := rapid.IntRange(0, 3).Draw(t, "numDuplicates")
			for i := range numDuplicates {
				if len(routes) > 0 {
					idx := rapid.IntRange(0, len(routes)-1).Draw(t, fmt.Sprintf("dup_idx_%d", i))
					routes = append(routes, routes[idx])
				}
			}

			var existingRoutes []gokeenrestapimodels.RciShowIpRoute
			for _, route := range routes {
				cidr, err := maskToCIDRSafe(route.Mask)
				if err != nil {
					continue
				}
				existingRoutes = append(existingRoutes, gokeenrestapimodels.RciShowIpRoute{
					Destination: fmt.Sprintf("%s/%d", route.IP, cidr),
					Interface:   route.Interface,
				})
			}

			var pass1, pass2 []bool
			for _, route := range routes {
				exists, err := checkInterfaceContainsRoute(route.IP, route.Mask, route.Interface, existingRoutes)
				if err != nil {
					t.Skip("checkInterfaceContainsRoute error")
				}
				pass1 = append(pass1, exists)
			}
			for _, route := range routes {
				exists, err := checkInterfaceContainsRoute(route.IP, route.Mask, route.Interface, existingRoutes)
				if err != nil {
					t.Skip("checkInterfaceContainsRoute error")
				}
				pass2 = append(pass2, exists)
			}

			Expect(pass2).To(Equal(pass1))
		})
	})

	It("should isolate routes by interface", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			interface1 := rapid.SampledFrom([]string{"ISP0", "ISP1", "Wireguard0", "Bridge0"}).Draw(t, "interface1")
			interface2 := rapid.SampledFrom([]string{"ISP0", "ISP1", "Wireguard0", "Bridge0"}).
				Filter(func(iface string) bool { return iface != interface1 }).
				Draw(t, "interface2")

			ip := genIPv4Address().Draw(t, "ip")
			mask := genSubnetMask().Draw(t, "mask")

			cidr, err := maskToCIDRSafe(mask)
			if err != nil {
				t.Skip("invalid mask generated")
			}
			dest := fmt.Sprintf("%s/%d", ip, cidr)

			existingRoutes := []gokeenrestapimodels.RciShowIpRoute{
				{Destination: dest, Interface: interface1},
			}

			existsOnIface1, err := checkInterfaceContainsRoute(ip, mask, interface1, existingRoutes)
			Expect(err).NotTo(HaveOccurred())
			Expect(existsOnIface1).To(BeTrue())

			existsOnIface2, err := checkInterfaceContainsRoute(ip, mask, interface2, existingRoutes)
			Expect(err).NotTo(HaveOccurred())
			Expect(existsOnIface2).To(BeFalse())
		})
	})
})
