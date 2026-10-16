package gokeenrestapi

import (
	"fmt"
	"slices"

	"github.com/noksa/gokeenapi/pkg/config"
)

// inspectExistingSplits checks that suffix groups look like a previous sorted
// partition or a partially applied update before they can be changed or removed.
// Keenetic does not expose a creator marker for object-groups, so names alone
// are never enough to authorize cleanup of an unexpected suffix.
func inspectExistingSplits(
	sources []config.DnsRoutingGroup,
	wantedDomains map[string][]string,
	wantedChunks map[string][]string,
	existingGroups map[string][]string,
	existingRoutes map[string]string,
) ([]config.DnsRoutingGroup, map[string]int, error) {
	var stale []config.DnsRoutingGroup
	oldCounts := make(map[string]int, len(sources))

	for _, source := range sources {
		desired, loaded := wantedDomains[source.Name]
		if !loaded {
			continue // An empty source must not remove existing router state.
		}

		baseDomains, baseExists := existingGroups[source.Name]
		oldCounts[source.Name] = len(baseDomains)
		baseInterface, baseRouteExists := existingRoutes[source.Name]
		indexed := make(map[int]string)
		for name, domains := range existingGroups {
			index := splitGroupIndex(source.Name, name)
			if index < 2 {
				continue
			}
			if !baseExists {
				return nil, nil, fmt.Errorf("group '%s': generated name '%s' exists without its base object-group", source.Name, name)
			}
			indexed[index] = name
			oldCounts[source.Name] += len(domains)
			interfaceID, routeExists := existingRoutes[name]
			if !routeExists {
				chunk, wanted := wantedChunks[name]
				if !wanted || len(baseDomains) != maxDomainsPerGroup || !allDomainsIn(domains, chunk) {
					return nil, nil, fmt.Errorf("group '%s': generated name '%s' has no route and cannot be safely resumed", source.Name, name)
				}
				continue
			}
			if interfaceID != source.InterfaceID && (!baseRouteExists || interfaceID != baseInterface) {
				return nil, nil, fmt.Errorf("group '%s': generated name '%s' already exists without a matching route on '%s' or the base group", source.Name, name, source.InterfaceID)
			}
			if _, wanted := wantedChunks[name]; !wanted {
				stale = append(stale, config.DnsRoutingGroup{Name: name, InterfaceID: interfaceID})
			}
		}
		for name, interfaceID := range existingRoutes {
			if splitGroupIndex(source.Name, name) >= 2 {
				if _, groupExists := existingGroups[name]; !groupExists {
					return nil, nil, fmt.Errorf("group '%s': generated route '%s' exists without an object-group on '%s'", source.Name, name, interfaceID)
				}
			}
		}

		for index := 2; index <= len(indexed)+1; index++ {
			if _, ok := indexed[index]; !ok {
				return nil, nil, fmt.Errorf("group '%s': noncontiguous generated group names; refusing automatic cleanup", source.Name)
			}
		}
		if len(indexed) == 0 {
			continue
		}

		// A complete previous partition remains recognizable even when the
		// current source legitimately removed domains. A partial update is also
		// safe to resume when every observed domain is in the current source.
		allExisting := slices.Clone(baseDomains)
		for index := 2; index <= len(indexed)+1; index++ {
			allExisting = append(allExisting, existingGroups[indexed[index]]...)
		}
		if !isSortedPartition(source.Name, indexed, allExisting, existingGroups) && !allDomainsIn(allExisting, desired) {
			return nil, nil, fmt.Errorf("group '%s': existing split does not match a previous sorted partition or current source; refusing to adopt it", source.Name)
		}
	}

	slices.SortFunc(stale, func(a, b config.DnsRoutingGroup) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return stale, oldCounts, nil
}

func allDomainsIn(domains, allowed []string) bool {
	set := make(map[string]bool, len(allowed))
	for _, domain := range allowed {
		set[domain] = true
	}
	for _, domain := range domains {
		if !set[domain] {
			return false
		}
	}
	return true
}

func isSortedPartition(base string, indexed map[int]string, all []string, groups map[string][]string) bool {
	slices.Sort(all)
	unique := slices.Compact(all)
	if len(unique) != len(all) {
		return false
	}
	for index := 1; index <= len(indexed)+1; index++ {
		name := base
		if index > 1 {
			name = indexed[index]
		}
		start := (index - 1) * maxDomainsPerGroup
		if start > len(all) {
			return false
		}
		end := min(start+maxDomainsPerGroup, len(all))
		actual := slices.Clone(groups[name])
		slices.Sort(actual)
		if !slices.Equal(actual, all[start:end]) {
			return false
		}
	}
	return true
}
