package serviceaccounts

import (
	"fmt"
	"sort"
)

// domainOrder defines the preferred display order for known domains.
// Unknown domains fall through to alphabetical order after the last entry.
var domainOrder = []string{"production", "development", "staging"}

// ServiceAccountsData is the page-level view model passed to the template.
type ServiceAccountsData struct {
	Projects []ProjectGroup
}

// ProjectGroup collects all domains belonging to a single Union project.
type ProjectGroup struct {
	Name    string
	Domains []DomainGroup
}

// DomainGroup collects all service accounts belonging to a single Union domain.
type DomainGroup struct {
	Name            string
	ServiceAccounts []ServiceAccount
}

// groupServiceAccounts converts the flat slice returned by the service layer
// into the nested Project > Domain > ServiceAccount structure the template
// expects. Projects and domains are sorted alphabetically for stable output.
func groupServiceAccounts(accounts []ServiceAccount) ServiceAccountsData {
	type domainKey struct{ project, domain string }

	// Preserve insertion order for projects; sort within each project.
	projectOrder := []string{}
	projectSeen := map[string]bool{}
	domainMap := map[domainKey][]ServiceAccount{}

	for _, sa := range accounts {
		if !projectSeen[sa.UnionProject] {
			projectSeen[sa.UnionProject] = true
			projectOrder = append(projectOrder, sa.UnionProject)
		}
		key := domainKey{sa.UnionProject, sa.UnionDomain}
		domainMap[key] = append(domainMap[key], sa)
	}

	sort.Strings(projectOrder)

	groups := make([]ProjectGroup, 0, len(projectOrder))
	for _, projectName := range projectOrder {
		// Collect and sort domains for this project.
		domainNames := []string{}
		domainSeen := map[string]bool{}
		for _, sa := range accounts {
			if sa.UnionProject == projectName && !domainSeen[sa.UnionDomain] {
				domainSeen[sa.UnionDomain] = true
				domainNames = append(domainNames, sa.UnionDomain)
			}
		}
		sort.Slice(domainNames, func(i, j int) bool {
			return domainRank(domainNames[i]) < domainRank(domainNames[j])
		})

		domains := make([]DomainGroup, 0, len(domainNames))
		for _, domainName := range domainNames {
			key := domainKey{projectName, domainName}
			sas := domainMap[key]
			sort.Slice(sas, func(i, j int) bool {
				return sas[i].K8sServiceAccount < sas[j].K8sServiceAccount
			})
			domains = append(domains, DomainGroup{
				Name:            domainName,
				ServiceAccounts: sas,
			})
		}

		groups = append(groups, ProjectGroup{
			Name:    projectName,
			Domains: domains,
		})
	}

	return ServiceAccountsData{Projects: groups}
}

// domainRank returns a sort key that places known domains first in the order
// defined by domainOrder, and sorts any unknown domains alphabetically after.
func domainRank(name string) string {
	for i, d := range domainOrder {
		if name == d {
			// Zero-pad so string comparison works correctly up to 999 known domains.
			return fmt.Sprintf("%03d", i)
		}
	}
	// Unknown domains sort after all known ones, then alphabetically among themselves.
	return fmt.Sprintf("%03d%s", len(domainOrder), name)
}
